package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	errors "github.com/lino-network/lino-errors"

	"github.com/coocood/freecache"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var (
	dbResponseTime  = 100 * time.Millisecond
	cacheableErrors = []errors.CodeType{errors.CodeServerInternalError}
)

type testSuite struct {
	suite.Suite
	redisConn   redis.UniversalClient
	inMemCache  *freecache.Cache
	cacheRepo   CacheRepository
	inMemCache2 *freecache.Cache
	cacheRepo2  CacheRepository
	mockRepo    dummyMock
}

type dummyMock struct {
	mock.Mock
}

// ReadThrough
func (_m *dummyMock) ReadThrough() (interface{}, errors.Error) {
	ret := _m.Called()
	// Emulate db response time
	time.Sleep(dbResponseTime)

	r0 := ret.Get(0)

	var r1 errors.Error
	if rf, ok := ret.Get(1).(func() errors.Error); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(errors.Error)
		}
	}

	return r0, r1
}

// WriteThrough
func (_m *dummyMock) WriteThrough() (interface{}, errors.Error) {
	ret := _m.Called()
	// Emulate db response time
	time.Sleep(dbResponseTime)

	r0 := ret.Get(0)

	var r1 errors.Error
	if rf, ok := ret.Get(1).(func() errors.Error); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(errors.Error)
		}
	}

	return r0, r1
}

func newTestSuite() *testSuite {
	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("127.0.0.1:6379"),
		DB:   10,
	})
	inMemCache := freecache.NewCache(1024 * 1024)
	cacheRepo, e := NewCacheRepo(redisClient, nil, nil, inMemCache, cacheableErrors, time.Second)
	if e != nil {
		panic(e)
	}
	inMemCache2 := freecache.NewCache(1024 * 1024)
	cacheRepo2, e := NewCacheRepo(redisClient, nil, nil, inMemCache2, cacheableErrors, time.Second)
	if e != nil {
		panic(e)
	}
	return &testSuite{
		redisConn:   redisClient,
		cacheRepo:   cacheRepo,
		inMemCache:  inMemCache,
		cacheRepo2:  cacheRepo2,
		inMemCache2: inMemCache2,
	}
}

func TestRepoTestSuite(t *testing.T) {
	suite.Run(t, newTestSuite())
}

func (suite *testSuite) BeforeTest(_, _ string) {
	suite.inMemCache.Clear()
	suite.inMemCache2.Clear()
	if err := suite.redisConn.FlushAll().Err(); err != nil {
		panic(err)
	}
}

func (suite *testSuite) AfterTest(_, _ string) {
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *testSuite) TearDownSuite() {
	suite.cacheRepo.Close()
	suite.cacheRepo2.Close()
}

func (suite *testSuite) decodeByte(bRes []byte, target interface{}) interface{} {
	dec := gob.NewDecoder(bytes.NewBuffer(bRes))
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr {
		// If target is not a pointer, create a pointer of target type and decode to it
		t := reflect.New(reflect.PtrTo(reflect.TypeOf(target)))
		e := dec.Decode(t.Interface())
		suite.NoError(e)
		// Dereference and return the underlying target
		return t.Elem().Elem().Interface()
	}
	e := dec.Decode(target)
	suite.NoError(e)
	// Use reflect to dereference an interface if it is pointer to array.
	// Should always be slice instead of array, but just in case.
	if value.Elem().Type().Kind() == reflect.Slice || value.Elem().Type().Kind() == reflect.Array {
		return reflect.ValueOf(target).Elem().Interface()
	}
	return target
}

func (suite *testSuite) TestPopulateCache() {
	queryKey := QueryKey("test")
	v := "testvalue"
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	// Second call should not hit db
	vget, err = suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	vredis := suite.redisConn.Get(store(queryKey)).Val()
	suite.Equal(v, suite.decodeByte([]byte(vredis), v))

	vinmem, e := suite.inMemCache.Get([]byte(store(queryKey)))
	suite.NoError(e)
	suite.Equal(v, suite.decodeByte(vinmem, v))

	// Second pod should not hit db either
	vget2, err := suite.cacheRepo2.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget2)

	vinmem2, e := suite.inMemCache2.Get([]byte(store(queryKey)))
	suite.NoError(e)
	suite.Equal(v, suite.decodeByte(vinmem2, v))
}

func (suite *testSuite) TestCachingArray() {
	queryKey := QueryKey("test")
	v := []string{"testvalue", "testvalue2", "testvalue3"}
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, &v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	// Second call should not hit db
	vget, err = suite.cacheRepo.Get(context.Background(), queryKey, &v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	vredis := suite.redisConn.Get(store(queryKey)).Val()
	suite.Equal(v, suite.decodeByte([]byte(vredis), v))

	vinmem, e := suite.inMemCache.Get([]byte(store(queryKey)))
	suite.NoError(e)
	suite.Equal(v, suite.decodeByte(vinmem, v))
}

func (suite *testSuite) TestCachingError() {
	queryKey := QueryKey("test")
	type a struct{}
	var v *a
	e := errors.NewErrorf(errors.CodeServerInternalError, "internal error")
	suite.mockRepo.On("ReadThrough").Return(v, e).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.Equal(e, err)
	suite.Equal(v, vget)

	// The second time should return err from cache not db
	vget, err = suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.Equal(e, err)
	suite.Equal(v, vget)
}

func (suite *testSuite) TestNotCachingError() {
	queryKey := QueryKey("test")
	v := ""
	// Not cacheable error
	e := errors.NewErrorf(errors.CodeAPIError, "internal error")
	// Should hit db twice
	suite.mockRepo.On("ReadThrough").Return(v, e).Twice()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.Equal(e, err)
	suite.Equal(v, vget)

	// The second time should return err from cache not db
	vget, err = suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.Equal(e, err)
	suite.Equal(v, vget)
}

func (suite *testSuite) TestConcurrentReadWait() {
	queryKey := QueryKey("test")
	v := "testvalue"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
			return suite.mockRepo.ReadThrough()
		}, false)
		suite.NoError(err)
		suite.Equal(v, vget)
	}()
	vget2, err := suite.cacheRepo2.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget2)
	wg.Wait()
}

func (suite *testSuite) TestConcurrentReadWaitTimeout() {
	queryKey := QueryKey("test")
	v := "testvalue"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
			return suite.mockRepo.ReadThrough()
		}, false)
		suite.NoError(err)
		suite.Equal(v, vget)
	}()
	// Make sure cache2 is called later and timeout is within db response time
	time.Sleep(dbResponseTime / 10)
	ctx, cancel := context.WithCancel(context.Background())
	// cancel the context
	cancel()
	_, err := suite.cacheRepo2.Get(ctx, queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	// Should get timeout error
	suite.Error(err)
	wg.Wait()
}

func (suite *testSuite) TestConcurrentReadAfterExpire() {
	queryKey := QueryKey("test")
	v := "testvalueold"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, time.Second, func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	var wg sync.WaitGroup

	// Wait for expire
	time.Sleep(time.Second * 1)
	// Change return
	newv := "testvaluenew"
	suite.mockRepo.On("ReadThrough").Return(newv, nil).Once()
	wg.Add(1)
	go func() {
		wg.Done()
		vget, err := suite.cacheRepo.Get(context.Background(), queryKey, newv, Normal.ToDuration(), func() (interface{}, errors.Error) {
			return suite.mockRepo.ReadThrough()
		}, false)
		suite.NoError(err)
		suite.Equal(newv, vget)
	}()
	// Make sure cache2 is called later and timeout is within db response time
	time.Sleep(dbResponseTime / 2)
	_, err = suite.cacheRepo2.Get(context.Background(), queryKey, newv, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	// The slower thread should return old cache value without wait
	suite.Equal(v, vget)

	time.Sleep(dbResponseTime)
	// Should get newv afterwards
	vget, err = suite.cacheRepo.Get(context.Background(), queryKey, newv, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(newv, vget)
	wg.Wait()
}

func (suite *testSuite) TestSet() {
	queryKey := QueryKey("test")
	v := "testvalueold"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	newv := "testvaluenew"
	suite.mockRepo.On("WriteThrough").Return(newv, nil).Once()
	vget, err = suite.cacheRepo.Set(context.Background(), []QueryKey{queryKey}, func() (interface{}, errors.Error) {
		return suite.mockRepo.WriteThrough()
	}, nil)
	suite.NoError(err)
	suite.Equal(newv, vget)

	// Wait for key to be deleted
	time.Sleep(500 * time.Millisecond)
	exist, e := suite.redisConn.Exists(store(queryKey), ttl(queryKey)).Result()
	suite.NoError(e)
	suite.EqualValues(0, exist)

	_, e = suite.inMemCache.Get([]byte(store(queryKey)))
	suite.Equal(freecache.ErrNotFound, e)

	suite.mockRepo.On("ReadThrough").Return(newv, nil).Once()
	vget, err = suite.cacheRepo.Get(context.Background(), queryKey, newv, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(newv, vget)
}

func (suite *testSuite) TestSetWithBlockChan() {
	var blockchan []chan struct{}
	queryKey := QueryKey("test")
	v := "testvalueold"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	newv := "testvaluenew"
	suite.mockRepo.On("WriteThrough").Return(newv, nil).Once()
	vget, err = suite.cacheRepo.Set(context.Background(), []QueryKey{queryKey}, func() (interface{}, errors.Error) {
		return suite.mockRepo.WriteThrough()
	}, &blockchan)
	suite.NoError(err)
	suite.Equal(newv, vget)

	// Wait for key to be deleted
	time.Sleep(500 * time.Millisecond)
	// Should not be deleted yet
	exist, e := suite.redisConn.Exists(store(queryKey), ttl(queryKey)).Result()
	suite.NoError(e)
	suite.EqualValues(2, exist)

	_, e = suite.inMemCache.Get([]byte(store(queryKey)))
	suite.NoError(e)

	for _, ch := range blockchan {
		ch <- struct{}{}
	}

	// Wait for key to be deleted
	time.Sleep(500 * time.Millisecond)
	// After sending signal should be deleted
	exist, e = suite.redisConn.Exists(store(queryKey), ttl(queryKey)).Result()
	suite.NoError(e)
	suite.EqualValues(0, exist)

	_, e = suite.inMemCache.Get([]byte(store(queryKey)))
	suite.Equal(freecache.ErrNotFound, e)
}

func (suite *testSuite) TestSetWithWriteBack() {
	queryKey := QueryKey("test")
	v := "testvalueold"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	newv := "testvaluenew"
	suite.mockRepo.On("WriteThrough").Return(newv, nil).Once()
	vget, err = suite.cacheRepo.SetWithWriteBack(context.Background(),
		map[QueryKey]CacheWriteBack{
			queryKey: CacheWriteBack{
				Value:  newv,
				Err:    nil,
				Expire: Normal.ToDuration(),
			},
		}, func() (interface{}, errors.Error) {
			return suite.mockRepo.WriteThrough()
		}, nil)
	suite.NoError(err)
	suite.Equal(newv, vget)

	// Wait for key to be populated
	time.Sleep(500 * time.Millisecond)
	vredis := suite.redisConn.Get(store(queryKey)).Val()
	suite.Equal(newv, suite.decodeByte([]byte(vredis), newv))

	vinmem, e := suite.inMemCache.Get([]byte(store(queryKey)))
	suite.NoError(e)
	suite.Equal(newv, suite.decodeByte(vinmem, newv))

	// Should not hit db
	vget, err = suite.cacheRepo.Get(context.Background(), queryKey, newv, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(newv, vget)
}

func (suite *testSuite) TestSetWithWriteBackWithBlockChan() {
	var blockchan []chan struct{}
	queryKey := QueryKey("test")
	v := "testvalueold"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	newv := "testvaluenew"
	suite.mockRepo.On("WriteThrough").Return(newv, nil).Once()
	vget, err = suite.cacheRepo.SetWithWriteBack(context.Background(),
		map[QueryKey]CacheWriteBack{
			queryKey: CacheWriteBack{
				Value:  newv,
				Err:    nil,
				Expire: Normal.ToDuration(),
			},
		}, func() (interface{}, errors.Error) {
			return suite.mockRepo.WriteThrough()
		}, &blockchan)
	suite.NoError(err)
	suite.Equal(newv, vget)

	// Wait for key to be deleted
	time.Sleep(500 * time.Millisecond)
	// Should not be updated yet
	vredis := suite.redisConn.Get(store(queryKey)).Val()
	suite.Equal(v, suite.decodeByte([]byte(vredis), v))

	vinmem, e := suite.inMemCache.Get([]byte(store(queryKey)))
	suite.NoError(e)
	suite.Equal(v, suite.decodeByte(vinmem, v))

	for _, ch := range blockchan {
		ch <- struct{}{}
	}

	// Wait for key to be populated
	time.Sleep(500 * time.Millisecond)
	vredis = suite.redisConn.Get(store(queryKey)).Val()
	suite.Equal(newv, suite.decodeByte([]byte(vredis), newv))

	vinmem, e = suite.inMemCache.Get([]byte(store(queryKey)))
	suite.NoError(e)
	suite.Equal(newv, suite.decodeByte(vinmem, newv))
}

func (suite *testSuite) TestInvalidateKeyAcrossPods() {
	queryKey := QueryKey("test")
	v := "testvalueold"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough").Return(v, nil).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	vget2, err := suite.cacheRepo2.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough()
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget2)

	vinmem, e := suite.inMemCache2.Get([]byte(store(queryKey)))
	suite.NoError(e)
	suite.Equal(v, suite.decodeByte(vinmem, v))

	vinmem, e = suite.inMemCache2.Get([]byte(store(queryKey)))
	suite.NoError(e)
	suite.Equal(v, suite.decodeByte(vinmem, v))

	newv := "testvaluenew"
	suite.mockRepo.On("WriteThrough").Return(newv, nil).Once()
	vget, err = suite.cacheRepo.Set(context.Background(), []QueryKey{queryKey}, func() (interface{}, errors.Error) {
		return suite.mockRepo.WriteThrough()
	}, nil)
	suite.NoError(err)
	suite.Equal(newv, vget)

	// Wait for key to be broadcasted
	time.Sleep(time.Second)
	exist, e := suite.redisConn.Exists(store(queryKey), ttl(queryKey)).Result()
	suite.NoError(e)
	suite.EqualValues(0, exist)

	_, e = suite.inMemCache.Get([]byte(store(queryKey)))
	suite.Equal(freecache.ErrNotFound, e)

	// check inmemcache of second pod is invalidated too
	_, e = suite.inMemCache2.Get([]byte(store(queryKey)))
	suite.Equal(freecache.ErrNotFound, e)
}
