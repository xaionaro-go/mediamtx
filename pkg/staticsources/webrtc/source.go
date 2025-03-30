// Package webrtc contains the WebRTC static source.
package webrtc

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/description"

	"github.com/xaionaro-go/mediamtx/pkg/conf"
	"github.com/xaionaro-go/mediamtx/pkg/defs"
	"github.com/xaionaro-go/mediamtx/pkg/logger"
	"github.com/xaionaro-go/mediamtx/pkg/protocols/tls"
	"github.com/xaionaro-go/mediamtx/pkg/protocols/webrtc"
	"github.com/xaionaro-go/mediamtx/pkg/protocols/whip"
	"github.com/xaionaro-go/mediamtx/pkg/stream"
)

// Source is a WebRTC static source.
type Source struct {
	ReadTimeout conf.Duration
	Parent      defs.StaticSourceParent
}

// Log implements logger.Writer.
func (s *Source) Log(level logger.Level, format string, args ...interface{}) {
	s.Parent.Log(level, "[WebRTC source] "+format, args...)
}

// Run implements StaticSource.
func (s *Source) Run(params defs.StaticSourceRunParams) error {
	s.Log(logger.Debug, "connecting")

	u, err := url.Parse(params.ResolvedSource)
	if err != nil {
		return err
	}

	u.Scheme = strings.ReplaceAll(u.Scheme, "whep", "http")

	tr := &http.Transport{
		TLSClientConfig: tls.ConfigForFingerprint(params.Conf.SourceFingerprint),
	}
	defer tr.CloseIdleConnections()

	client := whip.Client{
		HTTPClient: &http.Client{
			Timeout:   time.Duration(s.ReadTimeout),
			Transport: tr,
		},
		URL: u,
		Log: s,
	}

	err = client.Initialize(params.Context)
	if err != nil {
		return err
	}
	defer client.Close() //nolint:errcheck

	var stream *stream.Stream

	medias, err := webrtc.ToStream(client.PeerConnection(), &stream)
	if err != nil {
		return err
	}

	rres := s.Parent.SetReady(defs.PathSourceStaticSetReadyReq{
		Desc:               &description.Session{Medias: medias},
		GenerateRTPPackets: true,
	})
	if rres.Err != nil {
		return rres.Err
	}

	defer s.Parent.SetNotReady(defs.PathSourceStaticSetNotReadyReq{})

	stream = rres.Stream

	client.StartReading()

	return client.Wait(params.Context)
}

// APISourceDescribe implements StaticSource.
func (*Source) APISourceDescribe() defs.APIPathSourceOrReader {
	return defs.APIPathSourceOrReader{
		Type: "webRTCSource",
		ID:   "",
	}
}
