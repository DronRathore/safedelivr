package cassandra
import (
  "strconv"
  "github.com/gocql/gocql"
)

func Insert(table string, row map[string]interface{}) (bool, error) {
  var query = "INSERT INTO " + table + " ("
  var valStr = ") VALUES ("
  var elems int = len(row)
  var values []interface{}
  for column, val := range row {
    if elems == 1 {
      query = query + column
      valStr = valStr + "?)"
    } else {
      query = query + column + ","
      valStr = valStr + "?,"
    }
    values = append(values, val)
    elems = elems - 1
  }
  query = query + valStr
  err := Session.Query(query, values...).Exec()
  if err != nil {
    return false, err
  } else {
    return true, nil
  }
}

func Update(table string, where map[string]interface{}, row map[string]interface{}) (bool, error) {
  var query = "Update " + table + " SET "
  var count = len(row)
  var values []interface{}
  for column, val := range row {
    if count == 1 {
      query = query + column + "= ?"
      if len(where) != 0 {
        query = query + " WHERE "
      }
    } else {
      query = query + column + "= ?, "
    }
    count = count - 1
    values = append(values, val)
  }
  var whereStr string
  var clausesCount = len(where)
  for clause, val := range where {
   if count == 1 {
      whereStr = whereStr + clause + "= ?"
    } else {
      if whereStr != "" {
        whereStr = whereStr + " AND "
      }
      whereStr = whereStr + clause + "= ?"
    }
    values = append(values, val) 
    clausesCount = clausesCount - 1
  }
  query = query + whereStr
  err := Session.Query(query, values...).Exec()
  if err != nil {
    return false, err
  } else {
    return true, nil
  }
}

func Select(table string, columns string, where map[string]interface{}, limit int) (*gocql.Iter) {
  var query = "SELECT " + columns + " from " + table
  var values []interface{}
  if len(where) != 0 {
    query = query + " WHERE "
    var count = len(where)
    for column, val := range where {
      if count == 1 {
        query = query + column + "= ? "
        if limit >= 0 {
          query = query + " LIMIT " + strconv.Itoa(limit)
        }
      } else {
        query = query + column + "= ? AND "
      }
      values = append(values, val)
      count = count - 1
    }
  }
  if len(where) == 0 {
    return Session.Query(query).Consistency(gocql.One).Iter()
  }
  return Session.Query(query, values...).Consistency(gocql.One).Iter()
}

func Exec(query string) (bool, error) {
  err := Session.Query(query).Exec()
  if err != nil {
    return false, err
  }
  return true, nil
}