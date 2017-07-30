package models

type UserRedis struct {
  UUID string `json:uuid`
  Name string `json:name`
  Email string `json:email`
  Avatar_Url string `json:avatar_url`
  Company string `json:company`
  Location string `json:location`
}