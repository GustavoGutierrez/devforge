package dpf

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

const (
	poolDefaultSize = 2
	poolMaxSize     = 16
	poolSizeEnvVar  = "DEVFORGE_DPF_POOL_SIZE"
)

// Pool holds a free-list of Streamer instances and routes calls to whichever
// is currently idle. Callers block only when all instances are in use.
//
// Default pool size is 2; override with DEVFORGE_DPF_POOL_SIZE (1–16).
type Pool struct {
	clients chan Streamer
	size    int
}

// NewPool constructs a pool of *StreamClient instances for the given dpf binary.
// Size is resolved from DEVFORGE_DPF_POOL_SIZE (must be 1–16) or defaults to 2.
// Returns an error only when every StreamClient construction attempt fails.
func NewPool(binaryPath string) (*Pool, error) {
	size := resolvePoolSize()

	ch := make(chan Streamer, size)
	for i := 0; i < size; i++ {
		sc, err := NewStreamClient(binaryPath)
		if err != nil {
			// Close already-created clients before bailing out.
			close(ch)
			for existing := range ch {
				if c, ok := existing.(*StreamClient); ok {
					_ = c.Close()
				}
			}
			return nil, fmt.Errorf("dpf pool: failed to create stream client %d/%d: %w", i+1, size, err)
		}
		ch <- sc
	}

	return &Pool{clients: ch, size: size}, nil
}

// Size returns the number of Streamer instances in the pool.
func (p *Pool) Size() int {
	return p.size
}

// resolvePoolSize reads DEVFORGE_DPF_POOL_SIZE and returns a validated size.
func resolvePoolSize() int {
	raw := os.Getenv(poolSizeEnvVar)
	if raw == "" {
		return poolDefaultSize
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 || n > poolMaxSize {
		log.Printf("warning: %s=%q is invalid (must be 1–%d); using default %d",
			poolSizeEnvVar, raw, poolMaxSize, poolDefaultSize)
		return poolDefaultSize
	}

	return n
}

// acquire takes a Streamer from the free-list, blocking until one is available.
func (p *Pool) acquire() Streamer {
	return <-p.clients
}

// release returns a Streamer to the free-list.
func (p *Pool) release(s Streamer) {
	p.clients <- s
}

// Execute implements Streamer via the pool free-list.
func (p *Pool) Execute(job any) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.Execute(job)
}

// Crop implements Streamer.
func (p *Pool) Crop(job *CropJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.Crop(job)
}

// Rotate implements Streamer.
func (p *Pool) Rotate(job *RotateJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.Rotate(job)
}

// Watermark implements Streamer.
func (p *Pool) Watermark(job *WatermarkJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.Watermark(job)
}

// Adjust implements Streamer.
func (p *Pool) Adjust(job *AdjustJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.Adjust(job)
}

// AutoQuality implements Streamer.
func (p *Pool) AutoQuality(job *QualityJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.AutoQuality(job)
}

// Srcset implements Streamer.
func (p *Pool) Srcset(job *SrcsetJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.Srcset(job)
}

// Exif implements Streamer.
func (p *Pool) Exif(job *ExifJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.Exif(job)
}

// MarkdownToPDF implements Streamer.
func (p *Pool) MarkdownToPDF(job *MarkdownToPDFJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.MarkdownToPDF(job)
}

// VideoTranscode implements Streamer.
func (p *Pool) VideoTranscode(job *VideoTranscodeJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.VideoTranscode(job)
}

// VideoResize implements Streamer.
func (p *Pool) VideoResize(job *VideoResizeJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.VideoResize(job)
}

// VideoTrim implements Streamer.
func (p *Pool) VideoTrim(job *VideoTrimJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.VideoTrim(job)
}

// VideoThumbnail implements Streamer.
func (p *Pool) VideoThumbnail(job *VideoThumbnailJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.VideoThumbnail(job)
}

// VideoProfile implements Streamer.
func (p *Pool) VideoProfile(job *VideoProfileJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.VideoProfile(job)
}

// AudioTranscode implements Streamer.
func (p *Pool) AudioTranscode(job *AudioTranscodeJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.AudioTranscode(job)
}

// AudioTrim implements Streamer.
func (p *Pool) AudioTrim(job *AudioTrimJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.AudioTrim(job)
}

// AudioNormalize implements Streamer.
func (p *Pool) AudioNormalize(job *AudioNormalizeJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.AudioNormalize(job)
}

// AudioSilenceTrim implements Streamer.
func (p *Pool) AudioSilenceTrim(job *AudioSilenceTrimJob) (*JobResult, error) {
	c := p.acquire()
	defer p.release(c)
	return c.AudioSilenceTrim(job)
}
