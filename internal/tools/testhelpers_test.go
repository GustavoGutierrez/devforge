package tools_test

import (
	"dev-forge-mcp/internal/dpf"
)

// fakeStreamer is a test double for dpf.Streamer.
// Each method records the last job it received and returns the configured result/error.
type fakeStreamer struct {
	result *dpf.JobResult
	err    error

	// lastJob holds the most recent job passed to any method, for assertions.
	lastJob any
}

func (f *fakeStreamer) record(job any) (*dpf.JobResult, error) {
	f.lastJob = job
	return f.result, f.err
}

func (f *fakeStreamer) Close() error                                                   { return nil }
func (f *fakeStreamer) Execute(job any) (*dpf.JobResult, error)                        { return f.record(job) }
func (f *fakeStreamer) Crop(job *dpf.CropJob) (*dpf.JobResult, error)                  { return f.record(job) }
func (f *fakeStreamer) Rotate(job *dpf.RotateJob) (*dpf.JobResult, error)              { return f.record(job) }
func (f *fakeStreamer) Watermark(job *dpf.WatermarkJob) (*dpf.JobResult, error)        { return f.record(job) }
func (f *fakeStreamer) Adjust(job *dpf.AdjustJob) (*dpf.JobResult, error)              { return f.record(job) }
func (f *fakeStreamer) AutoQuality(job *dpf.QualityJob) (*dpf.JobResult, error)        { return f.record(job) }
func (f *fakeStreamer) Srcset(job *dpf.SrcsetJob) (*dpf.JobResult, error)              { return f.record(job) }
func (f *fakeStreamer) Exif(job *dpf.ExifJob) (*dpf.JobResult, error)                  { return f.record(job) }
func (f *fakeStreamer) MarkdownToPDF(job *dpf.MarkdownToPDFJob) (*dpf.JobResult, error) { return f.record(job) }
func (f *fakeStreamer) VideoTranscode(job *dpf.VideoTranscodeJob) (*dpf.JobResult, error) {
	return f.record(job)
}
func (f *fakeStreamer) VideoResize(job *dpf.VideoResizeJob) (*dpf.JobResult, error) {
	return f.record(job)
}
func (f *fakeStreamer) VideoTrim(job *dpf.VideoTrimJob) (*dpf.JobResult, error) { return f.record(job) }
func (f *fakeStreamer) VideoThumbnail(job *dpf.VideoThumbnailJob) (*dpf.JobResult, error) {
	return f.record(job)
}
func (f *fakeStreamer) VideoProfile(job *dpf.VideoProfileJob) (*dpf.JobResult, error) {
	return f.record(job)
}
func (f *fakeStreamer) AudioTranscode(job *dpf.AudioTranscodeJob) (*dpf.JobResult, error) {
	return f.record(job)
}
func (f *fakeStreamer) AudioTrim(job *dpf.AudioTrimJob) (*dpf.JobResult, error) { return f.record(job) }
func (f *fakeStreamer) AudioNormalize(job *dpf.AudioNormalizeJob) (*dpf.JobResult, error) {
	return f.record(job)
}
func (f *fakeStreamer) AudioSilenceTrim(job *dpf.AudioSilenceTrimJob) (*dpf.JobResult, error) {
	return f.record(job)
}

// successResult returns a minimal successful JobResult with one output at the given path.
func successResult(outputPath string) *dpf.JobResult {
	return &dpf.JobResult{
		Success:   true,
		Operation: "fake",
		ElapsedMs: 1,
		Outputs: []dpf.OutputFile{
			{
				Path:   outputPath,
				Format: "png",
				Width:  100,
				Height: 100,
			},
		},
	}
}
