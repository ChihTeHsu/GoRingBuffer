package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var Data = [...]int{86, 11, 73, 21, 93, 65, 33, 41, 58, 12}
const SIZE_OF_DATA = len(Data)	// Maximum size of buffer
const SIZE_OF_BUFFER = 3	// Maximum size of buffer
const NUM_OF_READER = 5	// Number of Reader to read from buffer

type Buffer struct {
	sync.Mutex
	cond *sync.Cond

	circularBuffer [SIZE_OF_BUFFER]int
	readIndex int // Index of the read pointer
	writeIndex int	// Index of the write pointer
	bufferLength int // Number of values currently in circular buffer
	counter uint32 // Count how many data Reader has been read
}

func (b *Buffer) Write(value int) {
	/*
		It may look like that the goroutine cond.Wait() is holding the lock whole time, but it is not.
		Internally cond.Wait() unlocks it while waiting and locks it again when it wakes up by other goroutine.
	*/
	b.cond.L.Lock()
	for b.bufferLength >= SIZE_OF_BUFFER {
		fmt.Printf("     buffer full, writer wait.....\n")
		b.cond.Wait() // IMPORTANT, once waked up from waiting it will use conditions that used be in above lock line !!!
		fmt.Printf("        writer released..... w/ buffer len:%d\n", b.bufferLength)
	}

	fmt.Printf("writer push the Data = %d w/ buffer len:%d\n", value, b.bufferLength)
	b.circularBuffer[b.writeIndex] = value // Write number to address of buffer index
	b.bufferLength++	 //	Increase buffer size after writing
	b.writeIndex++	 //	Increase writeIndex position to prepare for next write
	// If at last index in buffer, set writeIndex back to 0
	if b.writeIndex == SIZE_OF_BUFFER { b.writeIndex = 0 }

	fmt.Printf(" buffer filled to %d elements\n", b.bufferLength)
	b.cond.Broadcast()
	b.cond.L.Unlock()
}

func (b *Buffer) Read(worker int) bool {
	b.cond.L.Lock()

	// Use loop here let it checks buffer availability everytime it wakes up
	for b.bufferLength <= 0 {
		if atomic.LoadUint32(&b.counter) >= uint32(SIZE_OF_DATA) { // before waiting check first if task is end even buffer still empty
			b.cond.Broadcast() // when exist routine make sure broadcast to notify others
			b.cond.L.Unlock() // unlock before leave to let other routine wake up if any is waiting
			fmt.Printf("                                   reader %d finished\n", worker)
			return true
		}
		fmt.Printf("                                   reader %d wait....., w/ buffer len:%d\n", worker,b.bufferLength)
		b.cond.Wait()
	}

	fmt.Printf("                               reader %d pull Data= %d w/ buffer_len:%d,counter:%d\n",
		worker, b.circularBuffer[b.readIndex],b.bufferLength,atomic.LoadUint32(&b.counter))
	b.bufferLength--	 //	Decrease buffer size after reading
	b.readIndex++	 //	Increase readIndex position to prepare for next read
	// If at last index in buffer, set readIndex back to 0
	if b.readIndex == SIZE_OF_BUFFER { b.readIndex = 0 }
	fmt.Printf("                                    buffer decreased to %d elements\n", b.bufferLength)

	atomic.AddUint32(&b.counter, 1) // Increase counter by one
	fmt.Printf("                                     counter increased to %d\n",b.counter)
	b.cond.Broadcast()
	b.cond.L.Unlock()
	return false
}

func main() {
	var wg sync.WaitGroup
	buf := &Buffer{}
	buf.cond = sync.NewCond(buf)

	// This go routine keep sending until buffer is full and waiting for release
	wg.Add(1)
	go func() {
		for i:=0; i< SIZE_OF_DATA; i++ {
			buf.Write(Data[i])
		}
		wg.Done()
	}()

	// This one pull out data until buffer is empty, routine finish when counter=SIZE_OF_DATA
	for worker:=0; worker< NUM_OF_READER; worker++ {
		wg.Add(1)
		go func(worker int) {
			for {
				//time.Sleep(time.Second)
				if buf.Read(worker) { break }
			}
			wg.Done()
		}(worker)
	}

	wg.Wait()
}