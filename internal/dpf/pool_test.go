package dpf

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeStreamer is a test double that records call timestamps and sleeps
// a configurable duration to simulate real dpf work.
type fakeStreamer struct {
	sleepDur  time.Duration
	callCount atomic.Int64
}

func (f *fakeStreamer) Execute(job any) (*JobResult, error) {
	f.callCount.Add(1)
	time.Sleep(f.sleepDur)
	return &JobResult{Success: true, Operation: "fake"}, nil
}

func (f *fakeStreamer) Crop(job *CropJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) Rotate(job *RotateJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) Watermark(job *WatermarkJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) Adjust(job *AdjustJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) AutoQuality(job *QualityJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) Srcset(job *SrcsetJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) Exif(job *ExifJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) MarkdownToPDF(job *MarkdownToPDFJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) VideoTranscode(job *VideoTranscodeJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) VideoResize(job *VideoResizeJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) VideoTrim(job *VideoTrimJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) VideoThumbnail(job *VideoThumbnailJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) VideoProfile(job *VideoProfileJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) AudioTranscode(job *AudioTranscodeJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) AudioTrim(job *AudioTrimJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) AudioNormalize(job *AudioNormalizeJob) (*JobResult, error) {
	return f.Execute(job)
}

func (f *fakeStreamer) AudioSilenceTrim(job *AudioSilenceTrimJob) (*JobResult, error) {
	return f.Execute(job)
}

// newPoolFromStreamers builds a Pool from an existing slice of Streamer
// instances without spawning real dpf processes. For testing only.
func newPoolFromStreamers(streamers []Streamer) *Pool {
	ch := make(chan Streamer, len(streamers))
	for _, s := range streamers {
		ch <- s
	}
	return &Pool{clients: ch, size: len(streamers)}
}

// TestPoolParallelism verifies that a pool of size N can serve N concurrent
// calls in parallel, not serially (AC-10).
func TestPoolParallelism(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping parallelism timing test in -short mode")
	}

	const (
		poolSize    = 4
		numCalls    = 8
		callSleep   = 50 * time.Millisecond
		serialTotal = time.Duration(numCalls) * callSleep // 400ms
		maxAllowed  = serialTotal * 60 / 100              // 240ms (60% of serial)
	)

	streamers := make([]Streamer, poolSize)
	for i := range streamers {
		streamers[i] = &fakeStreamer{sleepDur: callSleep}
	}
	pool := newPoolFromStreamers(streamers)

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := pool.Execute(nil)
			if err != nil {
				t.Errorf("unexpected error from pool: %v", err)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("wall-clock: %v (serial baseline: %v, max allowed: %v)", elapsed, serialTotal, maxAllowed)
	if elapsed > maxAllowed {
		t.Errorf("pool of %d did not parallelize: elapsed %v > max %v (serial would be %v)",
			poolSize, elapsed, maxAllowed, serialTotal)
	}
}

// TestPoolBackpressure verifies that a pool of size 1 serializes calls
// (total wall-clock ≥ serialized baseline) without crashing (AC-10).
func TestPoolBackpressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping backpressure timing test in -short mode")
	}

	const (
		callSleep = 20 * time.Millisecond
		numCalls  = 3
		minTotal  = time.Duration(numCalls) * callSleep // at least 60ms if serial
	)

	pool := newPoolFromStreamers([]Streamer{
		&fakeStreamer{sleepDur: callSleep},
	})

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := pool.Execute(nil)
			if err != nil {
				t.Errorf("unexpected error from pool: %v", err)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("wall-clock: %v (expected ≥ %v for pool-of-1)", elapsed, minTotal)
	if elapsed < minTotal {
		t.Errorf("pool of 1 did not serialize: elapsed %v < expected %v", elapsed, minTotal)
	}
}
