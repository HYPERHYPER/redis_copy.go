package main

import (
  "fmt"
  "gopkg.in/redis.v5"
  "runtime"
  "time"
)

func testConnections(from, to *redis.Client) {
  fmt.Printf("from: %v to: %v\n", from, to)

  pong, err := to.Ping().Result()
  if err != nil {
    panic(err)
  }
  fmt.Println("redis labs:", pong, err)

  pong, err = from.Ping().Result()
  if err != nil {
    panic(err)
  }
  fmt.Println("elasticache:", pong, err)
}

func redisCopy(pattern string, counter chan int, done chan bool) {
  from := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
    // Password: "",
  })
  to := redis.NewClient(&redis.Options{
    Addr: "localhost:7379",
    // Password: "",
  })

  testConnections(from, to)

  // use SCAN to pull all keys
  // while there is API for pipelining this, it doesn't make
  // sense to me -- SCAN uses a cursor, and you have to pass
  // the cursor to the server on subsequent SCAN calls (which
  // I assume the redis library is doing under the hood)
  iter := from.Scan(0, pattern, 0).Iterator()

  for true {
    // read a chunk of keys into memory
    numKeys := 0
    var keys []string
    for iter.Next() {
      keys = append(keys, iter.Val())
      numKeys++
      if numKeys >= 7500 {
        break
      }
    }
    if err := iter.Err(); err != nil {
      panic(err)
    }
    num := len(keys)
    if num == 0 {
      // no more keys, we're done here
      break
    }

    // pipeline var used to read and write
    var pipe *redis.Pipeline

    // use DUMP to pull all values for this chunk of keys
    pipe = from.Pipeline()
    var dumps []*redis.StringCmd
    for i := range keys {
      dumps = append(dumps, pipe.Dump(keys[i]))
    }
    _, err := pipe.Exec()
    if err != nil {
      panic(err)
    }
    pipe.Close()

    // use RESTORE on "to" redis server for each key
    pipe = to.Pipeline()
    count := 0
    for i := range dumps {
      val := dumps[i].Val()
      _ = pipe.Restore(keys[i], 0, val)
      count++
    }
    _, err = pipe.Exec()
    if err != nil {
      // often happens due to duplicate keys, if you don't FLUSHDB first
      // also happens because of TCP timeouts. log it and move on.
      fmt.Println("RESTORE error(s):")
      fmt.Println(err)
      //panic(err)
    }
    pipe.Close()

    // report key chunk count back to main
    counter <- count
  }

  done <- true
}

func main() {
  // running 9 goroutines below, + main
  // this is dubiously effective
  runtime.GOMAXPROCS(10)

  total := 0
  numGorosDone := 0
  numGoroutines := 0
  start := time.Now()
  counter := make(chan int)
  done := make(chan bool)

  for i := 1; i < 10; i++ {
    pattern := fmt.Sprintf("feed_photo_ids:%v*", i)
    fmt.Println(pattern)
    go redisCopy(pattern, counter, done)
    numGoroutines++
  }

mainLoop:
  for true {
    select {
    case count := <-counter:
      total += count
      took := time.Since(start)
      fmt.Println("MAIN took", took, "for", total, "records")
    case <-done:
      numGorosDone++
      if numGorosDone == numGoroutines {
        break mainLoop
      }
    }
  }

  took := time.Since(start)
  fmt.Println("MAIN END took", took, "for", total, "records")
}
