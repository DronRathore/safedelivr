/*
  Database Initial setup helper
*/
package cassandra
import("fmt")
/*
  Add the alters, create and other modification of DB in the queries array
  and the server will execute them all on boot time
*/
func SetupDB(){
  var setupQueries = []string{
    // Keyspace definition
    "CREATE KEYSPACE IF NOT EXISTS SafeDelivr WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}",
    // Users table
    `Create Table IF NOT EXISTS safedelivr.users (
      user_id uuid,
      avatar_url varchar,
      name varchar,
      email varchar,
      auth_token varchar,
      api_key varchar,
      company varchar,
      location varchar,
      created_at timestamp,
      user_type int,
      PRIMARY KEY ((email))
    )`,
    // Indexes for User Table
    "Create Index IF NOT EXISTS user_api_key_index On safedelivr.users (api_key)",
    "Create Index IF NOT EXISTS user_id_index On safedelivr.users (user_id)",
    "Create Index IF NOT EXISTS user_location_index On safedelivr.users (location)",
    "Create Index IF NOT EXISTS user_auth_token_index On safedelivr.users (auth_token)",
    "Create Index IF NOT EXISTS user_type_index On safedelivr.users (user_type)",
    "Create Index IF NOT EXISTS user_company_index On safedelivr.users (company)",
    // todo: Adhere API calls to a quota
    `Create Table IF NOT EXISTS safedelivr.quota (
      user_id uuid,
      remaining counter,
      used counter,
      assigned counter,
      PRIMARY KEY ((user_id))
    )`,
    // Stats table
    `Create Table IF NOT EXISTS safedelivr.stats (
      user_id uuid,
      date timestamp,
      queued counter,
      success counter,
      failed counter,
      PRIMARY KEY ((user_id), date)
    )`,
    // Email batch table
    `Create Table IF NOT EXISTS safedelivr.batches (
      batch_id uuid,
      user_id uuid,
      subject varchar,
      created_at timestamp,
      last_updated timestamp,
      code varchar,
      description varchar,
      status varchar,
      reason varchar,
      options map<varchar, varchar>,
      PRIMARY KEY ((batch_id))
    )`,
    // Batch Indexes
    "Create Index IF NOT EXISTS user_id_index On safedelivr.batches (user_id)",
    "Create Index IF NOT EXISTS status_index On safedelivr.batches (status)",
    "Create Index IF NOT EXISTS created_at_index On safedelivr.batches (created_at)",
    // Log Table
    `Create Table IF NOT EXISTS safedelivr.logs (
      log_id uuid,
      batch_id uuid,
      user_id uuid,
      email varchar,
      state varchar,
      status map<varchar, boolean>,
      created_at timestamp,
      last_update timestamp,
      PRIMARY KEY ((log_id))  
    )`,
    // Log Tables Indexes
    "Create Index IF NOT EXISTS email_id_index On safedelivr.logs (email)",
    "Create Index IF NOT EXISTS batch_id_index On safedelivr.logs (batch_id)",
    "Create Index IF NOT EXISTS user_id_index On safedelivr.logs (user_id)",
    "Create Index IF NOT EXISTS state_index On safedelivr.logs (state)",
  }
  for _, query := range setupQueries {
    err := Session.Query(query).Exec()
    if err != nil {
      fmt.Println(err)
      panic("Cannot setup DB")
    }
  }
}