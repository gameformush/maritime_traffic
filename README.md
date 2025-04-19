## How to run 

```bash
docker compose up -d
```

starts two containers:

* web-jjnouxmwqg - starts server listening on port 8080

* test-ajnauxrwqb - runs all tests in the repository. Does not effect real server.

## Configuration

`PORT` env variable to change serving port

## Assumtions & edge cases

1. Position ship main logic is transactional
2. Red status does not change system state
3. Tower at 0,0 participates in collision detection
4. Max speed is 100. Ships which "jump" exceeding max speed are not over corrected.  
5. speed calculated linearly
6. Speed calculated using actual positions if avaliable otherwise predicts ship position using last known speed(depending on the time when prediction is happening)
7. Past predictions are allowed

## design 

Solution is based on simple http server and layered architecture:

1. HTTP RESTAPI layer - manager everything related to communication
2. Core layer - contains main logic with it's own data structures 

Solution uses InMemory storage. Which can handle 100_000_000 entities with good enough prefromance.

To handle bigger scale data could be stored in PostgreSQL.

## testing

* pkg/e2e/cases_test.go - e2e tests

* pkg/traffic/traffic_test.go - unit tests for the main logic

## performance & benchmark

* pkg/e2e/benchmarks_test.go - e2e benchmarks

* pkg/traffic/traffic_test.go - benchmark position ship logic 
