package dpf

// Streamer is the minimal contract that DevForge tools use to talk to dpf.
// Both *StreamClient and *Pool implement it. Tests can inject fakes.
//
// Only the methods actually called via tools.Server.DPF are included here;
// convenience wrappers on *Client (one-shot per-call) are not in scope.
type Streamer interface {
	// Close signals the underlying dpf process(es) to exit cleanly.
	// After Close returns, the Streamer must not be used again.
	Close() error

	// Execute sends any job payload to the dpf process and returns the result.
	Execute(job any) (*JobResult, error)

	// Image suite operations.
	Crop(job *CropJob) (*JobResult, error)
	Rotate(job *RotateJob) (*JobResult, error)
	Watermark(job *WatermarkJob) (*JobResult, error)
	Adjust(job *AdjustJob) (*JobResult, error)
	AutoQuality(job *QualityJob) (*JobResult, error)
	Srcset(job *SrcsetJob) (*JobResult, error)
	Exif(job *ExifJob) (*JobResult, error)

	// Document operations.
	MarkdownToPDF(job *MarkdownToPDFJob) (*JobResult, error)

	// Video operations.
	VideoTranscode(job *VideoTranscodeJob) (*JobResult, error)
	VideoResize(job *VideoResizeJob) (*JobResult, error)
	VideoTrim(job *VideoTrimJob) (*JobResult, error)
	VideoThumbnail(job *VideoThumbnailJob) (*JobResult, error)
	VideoProfile(job *VideoProfileJob) (*JobResult, error)

	// Audio operations.
	AudioTranscode(job *AudioTranscodeJob) (*JobResult, error)
	AudioTrim(job *AudioTrimJob) (*JobResult, error)
	AudioNormalize(job *AudioNormalizeJob) (*JobResult, error)
	AudioSilenceTrim(job *AudioSilenceTrimJob) (*JobResult, error)
}
