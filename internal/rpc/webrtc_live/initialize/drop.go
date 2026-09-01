package initialize

import (
	"strings"
	"sync/atomic"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

// rtpDropFactory is a deterministic benchmark-only fault injector. It is
// registered before NACK so retransmissions can use the responder cache.
type rtpDropFactory struct {
	everyN uint64
	onDrop func()
}

func (f *rtpDropFactory) NewInterceptor(string) (interceptor.Interceptor, error) {
	return &rtpDropInterceptor{everyN: f.everyN, onDrop: f.onDrop}, nil
}

type rtpDropInterceptor struct {
	interceptor.NoOp
	everyN uint64
	onDrop func()
	count  atomic.Uint64
}

func (i *rtpDropInterceptor) BindLocalStream(info *interceptor.StreamInfo, writer interceptor.RTPWriter) interceptor.RTPWriter {
	if i.everyN == 0 || !strings.HasPrefix(strings.ToLower(info.MimeType), "video/") {
		return writer
	}
	return interceptor.RTPWriterFunc(func(header *rtp.Header, payload []byte, attributes interceptor.Attributes) (int, error) {
		if i.count.Add(1)%i.everyN == 0 {
			if i.onDrop != nil {
				i.onDrop()
			}
			return len(payload), nil
		}
		return writer.Write(header, payload, attributes)
	})
}
