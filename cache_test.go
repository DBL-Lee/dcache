package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"reflect"
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
func (_m *dummyMock) ReadThrough(v interface{}, e errors.Error) (interface{}, errors.Error) {
	ret := _m.Called(v, e)
	// Emulate db response time
	time.Sleep(dbResponseTime)

	var r0 string
	if rf, ok := ret.Get(0).(func(interface{}, errors.Error) string); ok {
		r0 = rf(v, e)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(string)
		}
	}

	var r1 errors.Error
	if rf, ok := ret.Get(1).(func(interface{}, errors.Error) errors.Error); ok {
		r1 = rf(v, e)
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
	cacheRepo, e := NewCacheRepo(redisClient, nil, nil, inMemCache, cacheableErrors)
	if e != nil {
		panic(e)
	}
	inMemCache2 := freecache.NewCache(1024 * 1024)
	cacheRepo2, e := NewCacheRepo(redisClient, nil, nil, inMemCache2, cacheableErrors)
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

func (suite *testSuite) SetupTest() {
	suite.inMemCache.Clear()
	suite.inMemCache2.Clear()
	if err := suite.redisConn.FlushDB().Err(); err != nil {
		panic(err)
	}
	if err := suite.redisConn.ConfigResetStat().Err(); err != nil {
		panic(err)
	}
}

func (suite *testSuite) AfterTest(_, _ string) {
	suite.mockRepo.AssertExpectations(suite.T())
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
	suite.mockRepo.On("ReadThrough", v, nil).Return(v, nil).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough(v, nil)
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)

	// Second call should not hit db
	vget, err = suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough(v, nil)
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
		return suite.mockRepo.ReadThrough(v, nil)
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget2)

	vinmem2, e := suite.inMemCache2.Get([]byte(store(queryKey)))
	suite.NoError(e)
	suite.Equal(v, suite.decodeByte(vinmem2, v))
}

func (suite *testSuite) TestCachingError() {
	queryKey := QueryKey("test")
	v := ""
	err := errors.NewErrorf(errors.CodeServerInternalError, "internal error")
	suite.mockRepo.On("ReadThrough", v, err).Return(v, err).Once()
	vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough(v, nil)
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget)
}

func (suite *testSuite) TestConcurrentReadWait() {
	queryKey := QueryKey("test")
	v := "testvalue"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough", v, nil).Return(v, nil).Once()

	go func() {
		vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
			return suite.mockRepo.ReadThrough(v, nil)
		}, false)
		suite.NoError(err)
		suite.Equal(v, vget)
	}()
	vget2, err := suite.cacheRepo2.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough(v, nil)
	}, false)
	suite.NoError(err)
	suite.Equal(v, vget2)
}

func (suite *testSuite) TestConcurrentReadWaitTimeout() {
	queryKey := QueryKey("test")
	v := "testvalue"
	// Only one pod should hit db
	suite.mockRepo.On("ReadThrough", v, nil).Return(v, nil).Once()

	go func() {
		vget, err := suite.cacheRepo.Get(context.Background(), queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
			return suite.mockRepo.ReadThrough(v, nil)
		}, false)
		suite.NoError(err)
		suite.Equal(v, vget)
	}()
	// Make sure cache2 is called later and timeout is within db response time
	time.Sleep(dbResponseTime / 10)
	ctx, cancel := context.WithTimeout(context.Background(), dbResponseTime/2)
	defer cancel()
	_, err := suite.cacheRepo2.Get(ctx, queryKey, v, Normal.ToDuration(), func() (interface{}, errors.Error) {
		return suite.mockRepo.ReadThrough(v, nil)
	}, false)
	// Should get timeout error
	suite.Error(err)
}
