package helpers

import(
  "testing"
)

func TestIsMultipart(t *testing.T){
  if IsMultipart("lol; noboundary=kkk") != false {
    t.Log("Not a multipart header")
    t.Fail()
  }
  if IsMultipart("form-data; boundary=acdefawoeoeoeo") != true {
    t.Log("Is a multipart header")
    t.Fail()
  }
  if IsMultipart("form-data; boundaryies=acdefawoeoeoeo") != false {
    t.Log("Is a multipart header")
    t.Fail()
  }
}
