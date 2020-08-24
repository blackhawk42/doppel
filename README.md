# doppel
A utility to find duplicate files in a filesystem

# TODO
* Better help instructions.
* Introduce other hashes.
* Tests.
* Does concurrency help with speed, at all, or at least with this algorithm? Does the semaphore does anything? Find out.
* Race condition ocasionally appears. Example of a trace:
```
==================
WARNING: DATA RACE
Write at 0x00c0000d4068 by main goroutine:
  internal/race.Write()
      c:/go/src/internal/race/race.go:41 +0xf4
  sync.(*WaitGroup).Wait()
      c:/go/src/sync/waitgroup.go:128 +0xf5
  main.(*CollisionFinder).Result()
      D:/dev/doppel/collision_finder.go:112 +0x49
  main.main()
      D:/dev/doppel/main.go:112 +0x635

Previous read at 0x00c0000d4068 by goroutine 7:
  internal/race.Read()
      c:/go/src/internal/race/race.go:37 +0x1c9
  sync.(*WaitGroup).Add()
      c:/go/src/sync/waitgroup.go:71 +0x1dc
  main.(*CollisionFinder).Run.func1()
      D:/dev/doppel/collision_finder.go:49 +0x70

Goroutine 7 (finished) created at:
  main.(*CollisionFinder).Run()
      D:/dev/doppel/collision_finder.go:47 +0xac
  main.main()
      D:/dev/doppel/main.go:78 +0x4d2
==================
Found 1 data race(s)
```