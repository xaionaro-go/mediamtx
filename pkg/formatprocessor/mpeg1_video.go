package formatprocessor //nolint:dupl

import (
	"errors"
	"fmt"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtpmpeg1video"
	"github.com/pion/rtp"

	"github.com/xaionaro-go/mediamtx/pkg/unit"
)

// MPEG-1 video related parameters
var (
	MPEG1VideoDefaultConfig = []byte{
		0x00, 0x00, 0x01, 0xb3, 0x78, 0x04, 0x38, 0x35,
		0xff, 0xff, 0xe0, 0x18, 0x00, 0x00, 0x01, 0xb5,
		0x14, 0x4a, 0x00, 0x01, 0x00, 0x00,
	}
)

type formatProcessorMPEG1Video struct {
	udpMaxPayloadSize int
	format            *format.MPEG1Video
	encoder           *rtpmpeg1video.Encoder
	decoder           *rtpmpeg1video.Decoder
	randomStart       uint32
}

func newMPEG1Video(
	udpMaxPayloadSize int,
	forma *format.MPEG1Video,
	generateRTPPackets bool,
) (*formatProcessorMPEG1Video, error) {
	t := &formatProcessorMPEG1Video{
		udpMaxPayloadSize: udpMaxPayloadSize,
		format:            forma,
	}

	if generateRTPPackets {
		err := t.createEncoder()
		if err != nil {
			return nil, err
		}

		t.randomStart, err = randUint32()
		if err != nil {
			return nil, err
		}
	}

	return t, nil
}

func (t *formatProcessorMPEG1Video) createEncoder() error {
	t.encoder = &rtpmpeg1video.Encoder{
		PayloadMaxSize: t.udpMaxPayloadSize - 12,
	}
	return t.encoder.Init()
}

func (t *formatProcessorMPEG1Video) ProcessUnit(uu unit.Unit) error { //nolint:dupl
	u := uu.(*unit.MPEG1Video)

	// encode into RTP
	pkts, err := t.encoder.Encode(u.Frame)
	if err != nil {
		return err
	}
	u.RTPPackets = pkts

	for _, pkt := range u.RTPPackets {
		pkt.Timestamp += t.randomStart + uint32(u.PTS)
	}

	return nil
}

func (t *formatProcessorMPEG1Video) ProcessRTPPacket( //nolint:dupl
	pkt *rtp.Packet,
	ntp time.Time,
	pts int64,
	hasNonRTSPReaders bool,
) (unit.Unit, error) {
	u := &unit.MPEG1Video{
		Base: unit.Base{
			RTPPackets: []*rtp.Packet{pkt},
			NTP:        ntp,
			PTS:        pts,
		},
	}

	// remove padding
	pkt.Header.Padding = false
	pkt.PaddingSize = 0

	if pkt.MarshalSize() > t.udpMaxPayloadSize {
		return nil, fmt.Errorf("payload size (%d) is greater than maximum allowed (%d)",
			pkt.MarshalSize(), t.udpMaxPayloadSize)
	}

	// decode from RTP
	if hasNonRTSPReaders || t.decoder != nil {
		if t.decoder == nil {
			var err error
			t.decoder, err = t.format.CreateDecoder()
			if err != nil {
				return nil, err
			}
		}

		frame, err := t.decoder.Decode(pkt)
		if err != nil {
			if errors.Is(err, rtpmpeg1video.ErrNonStartingPacketAndNoPrevious) ||
				errors.Is(err, rtpmpeg1video.ErrMorePacketsNeeded) {
				return u, nil
			}
			return nil, err
		}

		u.Frame = frame
	}

	// route packet as is
	return u, nil
}