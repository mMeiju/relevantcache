package relevantcache

import (
	"io"
)

type option struct {
	name  string
	value interface{}
}

const (
	//optionNameSplitBufferSize = "split_buffer_size"
	optionNameSkipTLSVerify = "skip_tls_verify"
	optionNameDebugWriter   = "debug_log"
	optionNameScanCount     = "scan_count"
)

// func WithSplitBufferSize(size int64) option {
// 	return option{
// 		name:  optionNameSplitBufferSize,
// 		value: size,
// 	}
// }

func WithSkipTLSVerify(skip bool) option {
	return option{
		name:  optionNameSkipTLSVerify,
		value: skip,
	}
}

func WithDebugWriter(w io.Writer) option {
	return option{
		name:  optionNameDebugWriter,
		value: w,
	}
}

func WithScanCount(count int64) option {
	return option{
		name:  optionNameScanCount,
		value: count,
	}
}
