package relevantcache

type option struct {
	name  string
	value interface{}
}

const (
	//optionNameSplitBufferSize = "split_buffer_size"
	optionNameSkipTLSVerify = "skip_tls_verify"
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
