package redis

import (
  "fmt"
  "time"
  "config"
  redis "github.com/go-redis/redis"
)
type RedisStruct struct {
  client *redis.Client
}

var Redis *RedisStruct

func (r *RedisStruct) Connect() (error) {
  r.client = redis.NewClient(&redis.Options{
    Addr:     config.Configuration.Redis.Ip + ":" + config.Configuration.Redis.Port,
    DB:       config.Configuration.Redis.Db})
  _, err := r.client.Ping().Result()
  return err
}

func (r *RedisStruct) Get(key string) (string) {
  data, err := r.client.Get(key).Result()
  if err != nil {
    // shield main app from errors thrown from here
    fmt.Println(err)
    return ""
  }
  return data
}

func (r *RedisStruct) Set(key string, value string) error {
  return r.client.Set(key, value, 0).Err()
}

func (r *RedisStruct) SetTimed(key string, value string, duration time.Duration) error {
  return r.client.Set(key, value, duration).Err()
}

func (r *RedisStruct) Del(key string){
  r.client.Del(key)
}
func Init() *RedisStruct {
  Redis = &RedisStruct{}
  err := Redis.Connect()
  if err != nil {
    fmt.Println(err)
    panic("Cannot connect to the Redis server")
  }
  return Redis
}