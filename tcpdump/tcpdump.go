package tcpdump

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/gopacket"
	"github.com/google/gopacket/ip4defrag"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/pkg/errors"
)

var (
	// snaplen is number of bytes max to read per packet
	snaplen int32 = 65536
	// Name of decoder to use
	decoder = "Ethernet"
	// if true, do lazy decoding
	lazy = false
	// Print out packet dumps of decode errors, useful for checking decoders against live traffic
	printErrors = false
)

type Tcpdump struct {
	Interface string
	Ethernet  bool

	logger logr.Logger
}

func NewTcpdump(opt *Options, logger logr.Logger) *Tcpdump {
	return &Tcpdump{
		Interface: opt.Interface,
		Ethernet:  opt.Ethernet,
		logger:    logger,
	}
}

func (t *Tcpdump) Sniff(ctx context.Context, bpfFilter string) error {
	var pcapHandler *pcap.Handle
	var err error
	if pcapHandler, err = pcap.OpenLive(t.Interface, snaplen, true, pcap.BlockForever); err != nil {
		return errors.Wrap(err, "open live failed")
	}
	if err := pcapHandler.SetBPFFilter(bpfFilter); err != nil {
		return errors.Wrap(err, "set bpf fileter failed")
	}

	return t.dump(ctx, pcapHandler, false)
}

func (t *Tcpdump) dump(ctx context.Context, src gopacket.PacketDataSource, verbose bool) error {
	var dec gopacket.Decoder
	var ok bool
	if dec, ok = gopacket.DecodersByLayerName[decoder]; !ok {
		return errors.New(fmt.Sprintf("decoder %s not found", decoder))
	}

	source := gopacket.NewPacketSource(src, dec)
	source.Lazy = lazy
	source.NoCopy = true
	source.DecodeStreamsAsDatagrams = true

	count := 0
	bytes := int64(0)
	errorsNum := 0
	truncated := 0
	layertypes := map[gopacket.LayerType]int{}
	defragger := ip4defrag.NewIPv4Defragmenter()

	for {
		select {
		case <-ctx.Done():
			return nil
		case packet := <-source.Packets():
			count++
			bytes += int64(len(packet.Data()))

			// defrag the IPv4 packet is required
			ip4Layer := packet.Layer(layers.LayerTypeIPv4)
			if ip4Layer == nil {
				continue
			}
			ip4 := ip4Layer.(*layers.IPv4)
			l := ip4.Length

			newip4, err := defragger.DefragIPv4(ip4)
			if err != nil {
				t.logger.V(4).Info("Error while de-fragmenting", err)
			} else if newip4 == nil {
				// packet fragment, we don't have whole packet yet.
				continue
			}

			if newip4.Length != l {
				pb, ok := packet.(gopacket.PacketBuilder)
				if !ok {
					return errors.New("Not a PacketBuilder")
				}
				nextDecoder := newip4.NextLayerType()
				nextDecoder.Decode(newip4.Payload, pb)
			}

			if verbose {
				fmt.Println(packet.Dump())
			} else {
				fmt.Println(packet)
			}

			if !lazy {
				for _, layer := range packet.Layers() {
					layertypes[layer.LayerType()]++
				}
				if packet.Metadata().Truncated {
					truncated++
				}
				if errLayer := packet.ErrorLayer(); errLayer != nil {
					errorsNum++
					if printErrors {
						fmt.Println("Error:", errLayer.Error())
						fmt.Println("--- Packet ---")
						fmt.Println(packet.Dump())
					}
				}
			}
		}
	}
}
