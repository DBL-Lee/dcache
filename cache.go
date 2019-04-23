package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/go-redis/redis"
	errors "github.com/lino-network/lino-errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"
)

const (
	TTLSuffix                 = "_TTL"
	lockSuffix                = "_LOCK"
	minSleep                  = 50 * time.Millisecond
	MaxCacheTime              = time.Hour
	inMemCacheTime            = time.Second * 2
	redisCacheInvalidateTopic = "CacheInvalidatePubSub"
	maxInvalidate             = 100
	delimiter                 = "~|~"
)

var (
	nowFunc = time.Now
)

func SetNowFunc(f func() time.Time) { nowFunc = f }

type cacheError struct {
	Code errors.CodeType `json:"code"`
	Msg  string          `json:"msg"`
}

type ReadThroughFunc func() (interface{}, errors.Error)

type CacheRepository interface {
	Get(ctx context.Context, queryKey QueryKey, target interface{}, expire time.Duration, f ReadThroughFunc, noCache bool) (interface{}, errors.Error)
	Set(ctx context.Context, invalidateKeys []QueryKey, f ReadThroughFunc, block *[]chan struct{}) (interface{}, errors.Error)
	SetWithWriteBack(ctx context.Context, writeBack map[QueryKey]CacheWriteBack, f ReadThroughFunc, block *[]chan struct{}) (interface{}, errors.Error)
	Close()
}

// Client captures redis connection
type Client struct {
	primaryConn        redis.UniversalClient
	replicaConn        []redis.UniversalClient
	promCounter        *prometheus.CounterVec
	inMemCache         *freecache.Cache
	pubsub             *redis.PubSub
	id                 string
	invalidateKeys     map[string]struct{}
	invalidateMu       *sync.Mutex
	invalidateCh       chan struct{}
	cachableErrorTypes []errors.CodeType
}

// NewCacheRepo creates a new client for Cache connection
func NewCacheRepo(
	primaryClient redis.UniversalClient,
	replicaClient []redis.UniversalClient,
	counter *prometheus.CounterVec,
	inMemCache *freecache.Cache,
	cachableErrorTypes []errors.CodeType,
) (CacheRepository, errors.Error) {
	rand.Seed(nowFunc().UnixNano())
	id := uuid.NewV4()
	c := &Client{
		primaryConn:        primaryClient,
		replicaConn:        replicaClient,
		promCounter:        counter,
		id:                 id.String(),
		invalidateKeys:     make(map[string]struct{}),
		invalidateMu:       &sync.Mutex{},
		invalidateCh:       make(chan struct{}),
		inMemCache:         inMemCache,
		cachableErrorTypes: cachableErrorTypes,
	}
	if inMemCache != nil {
		c.pubsub = c.primaryConn.Subscribe(redisCacheInvalidateTopic)
		go c.aggregateSend()
		go c.listenKeyInvalidate()
	}
	return c, nil
}

func (c *Client) Close() {
	if c.pubsub != nil {
		c.pubsub.Unsubscribe()
		c.pubsub.Close()
	}
}

type QueryKey string

func (c *Client) getNoCache(ctx context.Context, queryKey QueryKey, expire time.Duration, f ReadThroughFunc) (interface{}, errors.Error) {
	if c.promCounter != nil {
		c.promCounter.WithLabelValues("MISS").Inc()
	}
	dbres, err := f()
	if err == nil {
		go func() {
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			e := enc.Encode(dbres)
			if e == nil {
				c.setKey(queryKey, buf.Bytes(), expire)
			}
		}()
	} else {
		if c.shouldCacheError(err) {
			go func() {
				cacheErr := cacheError{
					Code: err.CodeType(),
					Msg:  err.Error(),
				}
				var buf bytes.Buffer
				enc := gob.NewEncoder(&buf)
				e := enc.Encode(cacheErr)
				if e == nil {
					c.setKey(queryKey, buf.Bytes(), expire)
				} else {
					c.deleteKey(queryKey)
				}
			}()
		} else {
			c.deleteKey(queryKey)
		}
	}
	return dbres, err
}

func (c *Client) setKey(queryKey QueryKey, b []byte, expire time.Duration) {
	if c.primaryConn.Set(store(queryKey), b, MaxCacheTime).Err() == nil {
		c.primaryConn.Set(ttl(queryKey), strconv.FormatInt(nowFunc().UTC().Add(expire).Unix(), 10), expire)
	}
	if c.inMemCache != nil {
		c.inMemCache.Set([]byte(store(queryKey)), b, int(expire/time.Second))
		c.broadcastKeyInvalidate(queryKey)
	}
}

func (c *Client) deleteKey(queryKey QueryKey) {
	if e := c.primaryConn.Get(store(queryKey)).Err(); e != redis.Nil {
		// Delete key if error should not be cached
		c.primaryConn.Del(store(queryKey), ttl(queryKey))
	}
	if c.inMemCache != nil {
		c.inMemCache.Del([]byte(store(queryKey)))
		c.broadcastKeyInvalidate(queryKey)
	}
}

func (c *Client) broadcastKeyInvalidate(queryKey QueryKey) {
	c.invalidateMu.Lock()
	c.invalidateKeys[store(queryKey)] = struct{}{}
	l := len(c.invalidateKeys)
	c.invalidateMu.Unlock()
	if l == maxInvalidate {
		c.invalidateCh <- struct{}{}
	}
}

func (c *Client) aggregateSend() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-c.invalidateCh:
		}
		go func() {
			c.invalidateMu.Lock()
			if len(c.invalidateKeys) == 0 {
				c.invalidateMu.Unlock()
				return
			}
			toSend := c.invalidateKeys
			c.invalidateKeys = make(map[string]struct{})
			c.invalidateMu.Unlock()
			keys := make([]string, 0)
			for key := range toSend {
				keys = append(keys, key)
			}
			msg := c.id + delimiter + strings.Join(keys, delimiter)
			c.primaryConn.Publish(redisCacheInvalidateTopic, msg)
		}()
	}
}

func (c *Client) listenKeyInvalidate() {
	ch := c.pubsub.Channel()
	for {
		msg, ok := <-ch
		if !ok {
			return
		}
		payload := msg.Payload
		go func(payload string) {
			l := strings.Split(payload, delimiter)
			if len(l) < 2 {
				// Invalid payload
				log.Warn().Msgf("Received invalidate payload %s", payload)
				return
			}
			if l[0] == c.id {
				// Receive message from self
				return
			}
			// Invalidate key
			for _, key := range l[1:] {
				c.inMemCache.Del([]byte(key))
			}
		}(payload)
	}
}

func store(key QueryKey) string {
	return "{" + string(key) + "}"
}

func lock(key QueryKey) string {
	return store(key) + lockSuffix
}

func ttl(key QueryKey) string {
	return store(key) + TTLSuffix
}

func (c *Client) Get(ctx context.Context, queryKey QueryKey, target interface{}, expire time.Duration, f ReadThroughFunc, noCache bool) (interface{}, errors.Error) {
	if c.promCounter != nil {
		c.promCounter.WithLabelValues("TOTAL").Inc()
	}
	if noCache {
		return c.getNoCache(ctx, queryKey, expire, f)
	}
	readConn := c.primaryConn
	if len(c.replicaConn) > 0 {
		readConn = c.replicaConn[rand.Intn(len(c.replicaConn))]
	}

	var bRes []byte
	if c.inMemCache != nil {
		bRes, _ = c.inMemCache.Get([]byte(store(queryKey)))
	}
	if bRes == nil {
		var res, ttlRes string
		resList, e := readConn.MGet(store(queryKey), ttl(queryKey)).Result()
		if e == nil {
			if len(resList) != 2 {
				// Should never happen
				return nil, errors.NewErrorf(errors.CodeRedisFailedToGet, "get list %s not len 2", resList)
			}
			if resList[0] != nil {
				res, _ = resList[0].(string)
			}
			if resList[1] != nil {
				ttlRes, _ = resList[1].(string)
			}
		}
		if e != nil || resList[0] == nil {
			// Empty cache, obtain lock first to query db
			updated, _ := c.primaryConn.SetNX(lock(queryKey), "", time.Second*5).Result()
			if updated {
				return c.getNoCache(ctx, queryKey, expire, f)
			}
			// Did not obtain lock, sleep and retry to wait for update
		wait:
			for {
				select {
				case <-ctx.Done():
					return nil, errors.NewErrorf(errors.CodeRequestTimeout, "timeout")
				case <-time.After(minSleep):
					res, e = readConn.Get(store(queryKey)).Result()
					if e != nil {
						continue
					}
					break wait
				}
			}
		}
		bRes = []byte(res)

		var inMemExpire int
		// Tries to update ttl key if it doesn't exist
		if ttlRes == "" {
			// Key has expired, try to grab update lock
			updated, _ := c.primaryConn.SetNX(ttl(queryKey), strconv.FormatInt(nowFunc().UTC().Add(expire).Unix(), 10), expire).Result()
			if updated {
				// Got update lock
				return c.getNoCache(ctx, queryKey, expire, f)
			}
			inMemExpire = int(inMemCacheTime / time.Second)
		} else {
			// ttlRes should be unix time of expireAt
			t, e := strconv.ParseInt(ttlRes, 10, 64)
			if e != nil {
				t = nowFunc().UTC().Add(inMemCacheTime).Unix()
			}
			inMemExpire = int(t - nowFunc().UTC().Unix())
		}
		// Populate inMemCache
		if c.inMemCache != nil && inMemExpire > 0 {
			c.inMemCache.Set([]byte(store(queryKey)), bRes, inMemExpire)
		}
		if c.promCounter != nil {
			c.promCounter.WithLabelValues("REDIS HIT").Inc()
		}
	} else {
		if c.promCounter != nil {
			c.promCounter.WithLabelValues("INMEMCACHE HIT").Inc()
		}
	}
	if c.promCounter != nil {
		c.promCounter.WithLabelValues("HIT").Inc()
	}

	// check if value is err
	cachedErr := &cacheError{}
	dec := gob.NewDecoder(bytes.NewBuffer(bRes))
	e := dec.Decode(cachedErr)
	if e == nil && !cachedErr.isEmpty() {
		// Cast the ret to the nil pointer of same type if it is a pointer
		retReflect := reflect.ValueOf(target)
		if retReflect.Kind() == reflect.Ptr {
			value := reflect.New(retReflect.Type())
			return value.Elem().Interface(), errors.NewError(cachedErr.Code, cachedErr.Msg)
		}
		return target, errors.NewError(cachedErr.Code, cachedErr.Msg)
	}

	// check for actual value
	dec = gob.NewDecoder(bytes.NewBuffer(bRes))
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr {
		// If target is not a pointer, create a pointer of target type and decode to it
		t := reflect.New(reflect.PtrTo(reflect.TypeOf(target)))
		e = dec.Decode(t.Interface())
		if e != nil {
			return c.getNoCache(ctx, queryKey, expire, f)
		}
		// Dereference and return the underlying target
		return t.Elem().Elem().Interface(), nil
	}
	e = dec.Decode(target)
	if e != nil {
		return c.getNoCache(ctx, queryKey, expire, f)
	}
	// Use reflect to dereference an interface if it is pointer to array.
	// Should always be slice instead of array, but just in case.
	if value.Elem().Type().Kind() == reflect.Slice || value.Elem().Type().Kind() == reflect.Array {
		return reflect.ValueOf(target).Elem().Interface(), nil
	}
	return target, nil
}

func (c *Client) Set(ctx context.Context, invalidateKeys []QueryKey, f ReadThroughFunc, block *[]chan struct{}) (interface{}, errors.Error) {
	// if cond is passed, the process waits until cond is true before invalidating keys
	res, err := f()
	if err == nil {
		go func() {
			if block != nil {
				blockchan := make(chan struct{}, 1)
				*block = append(*block, blockchan)
				// Wait up to 30 seconds for unblock signal
				timer := time.NewTimer(30 * time.Second)
				defer timer.Stop()
				select {
				case <-blockchan:
				case <-timer.C:
					return
				}
			}
			for _, k := range invalidateKeys {
				c.deleteKey(k)
			}
		}()
	}
	return res, err
}

type CacheWriteBack struct {
	Value  interface{}
	Err    errors.Error
	Expire time.Duration
}

func (c *Client) SetWithWriteBack(ctx context.Context, writeBack map[QueryKey]CacheWriteBack, f ReadThroughFunc, block *[]chan struct{}) (interface{}, errors.Error) {
	res, err := f()
	if err == nil {
		go func() {
			if block != nil {
				blockchan := make(chan struct{}, 1)
				*block = append(*block, blockchan)
				// Wait up to 30 seconds for unblock signal
				timer := time.NewTimer(30 * time.Second)
				defer timer.Stop()
				select {
				case <-blockchan:
				case <-timer.C:
					return
				}
			}
			for k, v := range writeBack {
				c.getNoCache(ctx, k, v.Expire, func() (interface{}, errors.Error) { return v.Value, v.Err })
			}
		}()
	}
	return res, err
}

func (c *Client) shouldCacheError(err errors.Error) bool {
	for _, t := range c.cachableErrorTypes {
		if t == err.CodeType() {
			return true
		}
	}
	return false
}

func (c *cacheError) isEmpty() bool {
	return c.Code == 0 || c.Msg == ""
}
