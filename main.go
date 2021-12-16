package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const SIZE_OF_BUFFER=2	// Maximum size of buffer
const SIZE_OF_DATA=10	// Maximum size of buffer

type Buffer struct {
	sync.Mutex
	cond *sync.Cond

	Data []int
	bufferLength uint64 // Number of values currently in circular buffer
	counter uint32
}

func (b *Buffer) Write(data int) {
	/*
		It may look like that the goroutine cond.Wait() is holding the lock whole time, but it is not.
		Internally cond.Wait() unlocks it while waiting and locks it again when it wakes up by other goroutine.
	*/
	b.cond.L.Lock()
	fmt.Printf("writer want to push Data[%d] w/ buffer len:%d\n", data, b.bufferLength)
	for b.bufferLength >= SIZE_OF_BUFFER {
		fmt.Printf(" but buffer full, writer wait.....\n")
		b.cond.Wait() // IMPORTANT, once waked up from waiting it will use conditions that used be in above lock line !!!
		fmt.Printf("        writer released..... from:%d\n", b.bufferLength)
	}
	b.bufferLength ++
	fmt.Printf(" buffer filled to :%d\n", b.bufferLength)
	b.cond.Broadcast()
	b.cond.L.Unlock()
}

func (b *Buffer) Read(worker int) (finished bool) {
	b.cond.L.Lock()

	for (b.bufferLength <= 0)&&(atomic.LoadUint32(&b.counter) < SIZE_OF_DATA) {
		fmt.Printf("                                   reader %d wait....., w/ buffer:%d\n", worker,b.bufferLength)
		b.cond.Wait()
	}
	if atomic.LoadUint32(&b.counter) < SIZE_OF_DATA {
		idx := atomic.LoadUint32(&b.counter)
		fmt.Printf("                               reader %d pull Data[%d] w/ bufferLength:%d\n", worker, idx,b.bufferLength)
		b.bufferLength--
		fmt.Printf("                                    buffer decreased to :%d\n", b.bufferLength)

		//atomic.AddUint64(&readIndex, 1)
		atomic.AddUint32(&b.counter, 1)
		fmt.Printf("                                     done increased to %d\n",b.counter)
		finished = false
	} else {
		finished = true
	}
	b.cond.Broadcast()
	b.cond.L.Unlock()
	return finished
}

func main() {
	// 循环动态向 strList 切片中添加 SIZE_OF_DATA 个元素，并打印相关参数
	var Data []int
	for i := 0; i < SIZE_OF_DATA; i++ {
		Data = append(Data, i)
	}

	var wg sync.WaitGroup
	buf := &Buffer{}
	buf.cond = sync.NewCond(buf)

	// this go routine keep sending until buffer is full and waiting for release
	wg.Add(1)
	go func() {
		for i:=0; i< SIZE_OF_DATA; i++ {
			/*
			buf.cond.L.Lock()
			fmt.Printf("writer want to push Data[%d] w/ buffer len:%d\n", i, buf.bufferLength)
			for buf.bufferLength >= SIZE_OF_BUFFER {
				fmt.Printf(" but buffer full, writer wait.....\n")
				buf.cond.Wait() // IMPORTANT, once waked up from waiting it will use conditions that used be in above lock line !!!
				fmt.Printf("        writer released..... from:%d\n", buf.bufferLength)
			}
			buf.bufferLength ++
			fmt.Printf(" buffer filled to :%d\n", buf.bufferLength)
			buf.cond.Broadcast()
			buf.cond.L.Unlock()
			 */
			buf.Write(Data[i])
		}
		wg.Done()
	}()

	// this one pull out data and decrease counter
	//var readIndex uint64 = 0
	for j:=0; j<4; j++ {
		wg.Add(1)
		go func(worker int) {
			for {
				//time.Sleep(time.Second)
				/*
				buf.cond.L.Lock()
				for buf.bufferLength <= 0 {
					fmt.Printf("                                   reader %d wait....., w/ buffer:%d\n", worker,buf.bufferLength)
					buf.cond.Wait()
					if atomic.LoadUint32(&buf.done) >= SIZE_OF_DATA { break }
				}
				idx := atomic.LoadUint32(&buf.done)
				fmt.Printf("                               reader %d pull Data[%d] w/ bufferLength:%d\n", worker, idx,buf.bufferLength)
				buf.bufferLength--
				fmt.Printf("                                    buffer decreased to :%d\n", buf.bufferLength)

				//atomic.AddUint64(&readIndex, 1)
				atomic.AddUint32(&buf.done, 1)
				fmt.Printf("                                     done increased to %d\n",buf.done)

				buf.cond.Broadcast()
				buf.cond.L.Unlock()

				if atomic.LoadUint32(&buf.done) >= SIZE_OF_DATA {
					break
				}
				 */
				if buf.Read(worker) {
					break
				}
			}
			wg.Done()
		}(j)
	}

	wg.Wait()
}