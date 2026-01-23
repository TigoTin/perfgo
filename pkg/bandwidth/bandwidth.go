package bandwidth

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"perfgo/pkg/protocol"
	"perfgo/pkg/utils"
)

const (
	// DefaultBufferSize 默认缓冲区大小
	DefaultBufferSize = 64 * 1024 // 64KB
	// TestDuration 默认测试持续时间
	TestDuration = 10 * time.Second
	// DataChunkSize 数据块大小
	DataChunkSize = 32 * 1024 // 32KB
)

// BandwidthTester 带宽测试器
type BandwidthTester struct {
	BufferSize int
	Duration   time.Duration
}

// NewBandwidthTester 创建新的带宽测试器
func NewBandwidthTester() *BandwidthTester {
	return &BandwidthTester{
		BufferSize: DefaultBufferSize,
		Duration:   TestDuration,
	}
}

// UploadTest 执行上传速度测试（从客户端到服务器）
func (bt *BandwidthTester) UploadTest(conn net.Conn, threads int) error {
	fmt.Printf("开始上传速度测试，线程数: %d...\n", threads)

	startTime := time.Now()
	endTime := startTime.Add(bt.Duration)

	var totalBytes int64
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 准备数据块
	data := make([]byte, DataChunkSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// 如果线程数为1，则使用单线程模式
	if threads <= 1 {
		// 单线程上传测试
		ctx, cancel := context.WithTimeout(context.Background(), bt.Duration)
		defer cancel()

		done := make(chan error, 1)

		// 启动发送协程
		go func() {
			defer close(done)

			threadBytes := int64(0)
			for {
				select {
				case <-ctx.Done():
					mu.Lock()
					totalBytes = threadBytes
					mu.Unlock()
					done <- nil
					return
				default:
					n, err := conn.Write(data)
					if err != nil {
						mu.Lock()
						totalBytes = threadBytes
						mu.Unlock()
						done <- err
						return
					}

					threadBytes += int64(n)

					// 检查是否已达到结束时间
					if time.Now().After(endTime) {
						mu.Lock()
						totalBytes = threadBytes
						mu.Unlock()
						done <- nil
						return
					}
				}
			}
		}()

		// 等待测试完成
		err := <-done
		if err != nil {
			return fmt.Errorf("upload test error: %v", err)
		}
	} else {
		// 多线程上传测试
		for i := 0; i < threads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				threadBytes := int64(0)
				ctx, cancel := context.WithTimeout(context.Background(), bt.Duration)
				defer cancel()

				for {
					select {
					case <-ctx.Done():
						goto finish
					default:
						n, err := conn.Write(data)
						if err != nil {
							fmt.Printf("上传线程 %d 错误: %v\n", i, err)
							goto finish
						}

						threadBytes += int64(n)

						// 检查是否已达到结束时间
						if time.Now().After(endTime) {
							goto finish
						}
					}
				}

			finish:
				mu.Lock()
				totalBytes += threadBytes
				mu.Unlock()
			}()
		}

		// 等待所有协程完成
		wg.Wait()
	}

	elapsed := time.Since(startTime)
	throughput := float64(totalBytes) / elapsed.Seconds()

	fmt.Printf("上传速度测试完成:\n")
	fmt.Printf("  测试时长: %.2fs\n", elapsed.Seconds())
	fmt.Printf("  线程数: %d\n", threads)
	fmt.Printf("  发送字节数: %s\n", utils.FormatBytes(totalBytes))
	fmt.Printf("  速度: %s\n", utils.FormatSpeedDetailed(throughput))

	// 在带宽测试完成后，清理连接上的任何残留数据
	// 设置一个短超时来读取可能的残留数据
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			// 如果是超时错误，表示没有更多数据
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			// 其他错误则跳出
			break
		}
		// 如果读取到数据，继续读取直到没有更多数据
		if n == 0 {
			break
		}
	}
	// 重置读取超时
	conn.SetReadDeadline(time.Time{})

	return nil
}

// DownloadTest 执行下载速度测试（从服务器到客户端）
func (bt *BandwidthTester) DownloadTest(conn net.Conn, threads int) error {
	fmt.Printf("开始下载速度测试，线程数: %d...\n", threads)

	startTime := time.Now()
	endTime := startTime.Add(bt.Duration)

	var totalBytes int64
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 准备缓冲区
	buffer := make([]byte, bt.BufferSize)

	// 如果线程数为1，则使用单线程模式
	if threads <= 1 {
		// 单线程下载测试
		ctx, cancel := context.WithTimeout(context.Background(), bt.Duration)
		defer cancel()

		done := make(chan error, 1)

		// 启动接收协程
		go func() {
			defer close(done)

			threadBytes := int64(0)
			for {
				select {
				case <-ctx.Done():
					mu.Lock()
					totalBytes = threadBytes
					mu.Unlock()
					done <- nil
					return
				default:
					n, err := conn.Read(buffer)
					if err != nil {
						// 不再处理EOF，因为带宽测试期间连接不应关闭
						mu.Lock()
						totalBytes = threadBytes
						mu.Unlock()
						done <- err
						return
					}

					threadBytes += int64(n)

					// 检查是否已达到结束时间
					if time.Now().After(endTime) {
						mu.Lock()
						totalBytes = threadBytes
						mu.Unlock()
						done <- nil
						return
					}
				}
			}
		}()

		// 等待测试完成
		err := <-done
		if err != nil {
			return fmt.Errorf("download test error: %v", err)
		}
	} else {
		// 多线程下载测试
		for i := 0; i < threads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				threadBytes := int64(0)
				ctx, cancel := context.WithTimeout(context.Background(), bt.Duration)
				defer cancel()

				for {
					select {
					case <-ctx.Done():
						goto finish
					default:
						n, err := conn.Read(buffer)
						if err != nil {
							// 不再处理EOF，因为带宽测试期间连接不应关闭
							fmt.Printf("下载线程 %d 错误: %v\n", i, err)
							goto finish
						}

						threadBytes += int64(n)

						// 检查是否已达到结束时间
						if time.Now().After(endTime) {
							goto finish
						}
					}
				}

			finish:
				mu.Lock()
				totalBytes += threadBytes
				mu.Unlock()
			}()
		}

		// 等待所有协程完成
		wg.Wait()
	}

	elapsed := time.Since(startTime)
	throughput := float64(totalBytes) / elapsed.Seconds()

	fmt.Printf("下载速度测试完成:\n")
	fmt.Printf("  测试时长: %.2fs\n", elapsed.Seconds())
	fmt.Printf("  线程数: %d\n", threads)
	fmt.Printf("  接收字节数: %s\n", utils.FormatBytes(totalBytes))
	fmt.Printf("  速度: %s\n", utils.FormatSpeedDetailed(throughput))

	// 在带宽测试完成后，清理连接上的任何残留数据
	// 设置一个短超时来读取可能的残留数据
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			// 如果是超时错误，表示没有更多数据
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			// 其他错误则跳出
			break
		}
		// 如果读取到数据，继续读取直到没有更多数据
		if n == 0 {
			break
		}
	}
	// 重置读取超时
	conn.SetReadDeadline(time.Time{})

	return nil
}

// ServerHandle 处理来自客户端的带宽测试请求
func (bt *BandwidthTester) ServerHandle(conn net.Conn, msg *protocol.Message) error {
	testType, ok := msg.Payload["test_type"].(string)
	if !ok {
		return fmt.Errorf("missing test_type in payload")
	}

	threads, ok := msg.Payload["threads"].(float64)
	if !ok {
		threads = 1 // 默认为1个线程
	}

	// 检查是否提供了测试持续时间
	duration, ok := msg.Payload["duration"].(float64)
	if !ok {
		duration = 10 // 默认10秒
	}

	bt.Duration = time.Duration(duration) * time.Second

	switch testType {
	case "upload":
		// 服务器处理下载测试（接收客户端发送的数据）
		return bt.DownloadTest(conn, int(threads))
	case "download":
		// 服务器处理上传测试（向客户端发送数据）
		return bt.UploadTest(conn, int(threads))
	default:
		return fmt.Errorf("unknown test type: %s", testType)
	}
}
