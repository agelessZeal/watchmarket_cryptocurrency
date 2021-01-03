package rediscache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/trustwallet/watchmarket/pkg/watchmarket"
	"github.com/trustwallet/watchmarket/redis"
	"testing"
	"time"
)

func TestInstance_GetCharts_notOutdated(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()

	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)
	seedDbCharts(t, i)

	data, err := i.getIntervalKey("testKEY", 1, context.Background())
	assert.NotNil(t, data)
	assert.Nil(t, err)

	d, err := i.GetWithTime("testKEY", 0, context.Background())
	assert.Nil(t, err)
	ch := watchmarket.Chart{}
	err = json.Unmarshal(d, &ch)
	assert.Nil(t, err)
	assert.Equal(t, makeChartDataMock(), ch)
}

func TestInstance_GetCharts_CachingDataWasEmpty(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()

	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)

	res, err := json.Marshal([]CachedInterval{{Timestamp: 0, Duration: 100000, Key: "A"}})
	assert.Nil(t, err)
	assert.NotNil(t, res)

	err = i.redis.Set("testKEY", res, watchmarket.UnixToDuration(1000), context.Background())
	assert.Nil(t, err)

	d, err := i.GetWithTime("testKEY", 10000, context.Background())
	assert.Equal(t, "cache is not valid", err.Error())
	assert.Equal(t, "", string(d))
}

func TestInstance_GetCharts_notExistingKey(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()
	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)

	seedDbCharts(t, i)

	d, err := i.GetWithTime("testKEY+1", 1, context.Background())
	assert.Equal(t, "Not found", err.Error())
	assert.Equal(t, "", string(d))
}

func TestInstance_GetCharts_Outdated(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()
	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)

	d, err := i.GetWithTime("testKEY", 100000, context.Background())
	assert.NotNil(t, err)
	ch := watchmarket.Chart{}
	err = json.Unmarshal(d, &ch)
	assert.Equal(t, watchmarket.Chart{}, ch)
	assert.NotNil(t, err)
}

func TestInstance_GetCharts_OutdatedCacheIsNotReturned(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()

	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)

	d, err := i.GetWithTime("testKEY", 100000, context.Background())
	assert.NotNil(t, err)
	ch := watchmarket.Chart{}
	err = json.Unmarshal(d, &ch)
	assert.Equal(t, watchmarket.Chart{}, ch)
	assert.NotNil(t, err)

	res, err := i.redis.Get("testKEY", context.Background())
	assert.NotNil(t, err)
	assert.Nil(t, res)
}

func TestInstance_GetCharts_ValidCacheIsReturned(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()

	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)

	seedDbCharts(t, i)

	d, err := i.GetWithTime("testKEY", 100, context.Background())
	assert.Nil(t, err)
	ch := watchmarket.Chart{}
	err = json.Unmarshal(d, &ch)
	assert.Equal(t, makeChartDataMock(), ch)
	assert.Nil(t, err)

	res, err := i.redis.Get("data_key", context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, res)
}

func TestInstance_GetCharts_StartTimeIsEarlierThatWasCached(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()

	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)

	d, err := i.GetWithTime("testKEY", -1, context.Background())
	assert.NotNil(t, err)
	ch := watchmarket.Chart{}
	err = json.Unmarshal(d, &ch)
	assert.Equal(t, watchmarket.Chart{}, ch)
	assert.NotNil(t, err)

	res, err := i.redis.Get("testKEY", context.Background())
	assert.NotNil(t, err)
	assert.Nil(t, res)

	// emulate that cache was created
	seedDbCharts(t, i)

	d2, err := i.GetWithTime("testKEY", 100, context.Background())
	assert.Nil(t, err)
	ch2 := watchmarket.Chart{}
	err = json.Unmarshal(d2, &ch2)
	assert.Equal(t, makeChartDataMock(), ch2)
	assert.Nil(t, err)

	resTwo, err := i.redis.Get("data_key", context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, resTwo)
}

func TestInstance_GetCharts(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()

	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)

	err = r.Set("data_key", []byte{0, 1, 2}, time.Minute, context.Background())
	assert.Nil(t, err)

	err = i.updateInterval("testKEY", CachedInterval{
		Timestamp: 0,
		Duration:  1000,
		Key:       "data_key",
	}, context.Background())
	assert.Nil(t, err)

	d, err := i.GetWithTime("testKEY", 1, context.Background())
	assert.Nil(t, err)
	ch := watchmarket.Chart{}
	err = json.Unmarshal(d, &ch)
	assert.NotNil(t, err)
	assert.Equal(t, watchmarket.Chart{}, ch)

	res, err := i.redis.Get("testKEY", context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, res)
}

func TestInstance_SaveCharts(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()

	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)
	res, err := json.Marshal(makeChartDataMock())
	assert.Nil(t, err)
	err = i.SetWithTime("testKEY", res, 0, context.Background())
	assert.Nil(t, err)

	res, err = i.redis.Get("xQNa0B7ITYf1gJY0dGG3fabGPic=", context.Background())
	assert.Nil(t, err)
	mocked, err := makeRawDataMockCharts()
	assert.Nil(t, err)
	assert.Equal(t, mocked, res)
	assert.Nil(t, err)
}

func TestProvider_Mixed(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()

	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)
	res, err := json.Marshal(makeChartDataMock())
	assert.Nil(t, err)
	err = i.SetWithTime("testKEY", res, 0, context.Background())
	assert.Nil(t, err)

	d, err := i.GetWithTime("testKEY", 100, context.Background())
	ch := watchmarket.Chart{}
	_ = json.Unmarshal(d, &ch)
	assert.Equal(t, makeChartDataMock(), ch)
	assert.Nil(t, err)

	_, err = i.GetWithTime("testKEY", 10001, context.Background())
	assert.NotNil(t, err)
	assert.Equal(t, "no suitable intervals", err.Error())
}

func TestInstance_SaveCharts_DataIsEmpty(t *testing.T) {
	s := setupRedis(t)
	defer s.Close()

	r, err := redis.Init(fmt.Sprintf("redis://%s", s.Addr()))
	assert.Nil(t, err)

	i := Init(r, time.Second*1000)
	assert.NotNil(t, i)

	err = i.SetWithTime("testKEY", nil, 0, context.Background())
	assert.Equal(t, "data is empty", err.Error())
	d, err := i.GetWithTime("testKEY", 0, context.Background())
	assert.NotNil(t, err)
	ch := watchmarket.Chart{}
	err = json.Unmarshal(d, &ch)
	assert.NotNil(t, err)
	assert.Equal(t, watchmarket.Chart{}, ch)
}

func seedDbCharts(t *testing.T, instance Instance) {
	rawData, err := makeRawDataMockCharts()
	assert.NotNil(t, rawData)
	assert.Nil(t, err)
	_ = instance.updateInterval("testKEY", CachedInterval{
		Timestamp: 0,
		Duration:  1000,
		Key:       "data_key",
	}, context.Background())
	_ = instance.redis.Set("data_key", rawData, watchmarket.UnixToDuration(1000), context.Background())

}

func makeRawDataMockCharts() ([]byte, error) {
	rawData, err := json.Marshal(makeChartDataMock())
	if err != nil {
		return nil, err
	}

	return rawData, nil
}

func makeChartDataMock() watchmarket.Chart {
	price := watchmarket.ChartPrice{
		Price: 100000,
		Date:  0,
	}

	prices := make([]watchmarket.ChartPrice, 0)
	prices = append(prices, price)
	prices = append(prices, price)

	return watchmarket.Chart{
		Prices: prices,
		Error:  "",
	}
}
