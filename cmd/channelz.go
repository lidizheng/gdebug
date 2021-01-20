package cmd

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"gdebug/transport"

	"github.com/dustin/go-humanize"
	"github.com/golang/protobuf/ptypes"
	timestamppb "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/spf13/cobra"
	zpb "google.golang.org/grpc/channelz/grpc_channelz_v1"
)

// Applies to channels and subchannels
type channelInspectEntry struct {
	ID     int64
	Target string
	State  string
	// The number of calls started.
	Calls    int64
	Resolver string
	LBPolicy string
	// A human readable string.
	CreatedTime string
}

type channelTraceEventInspectEntry struct {
	Severity    string
	Time        string
	ChildRef    string
	Description string
}

type socketInspectEntry struct {
	SocketID int64
	// LocalAddress -> RemoteAddress.
	Address string
	// The number of streams started.
	Streams int64
	// The number of messages started.
	Messages int64
}

// The table formater
var w = tabwriter.NewWriter(os.Stdout, 10, 0, 3, ' ', 0)

func prettyTime(ts *timestamppb.Timestamp) string {
	t, _ := ptypes.Timestamp(ts)
	return humanize.Time(t)
}

func prettySeverity(s zpb.ChannelTraceEvent_Severity) string {
	return zpb.ChannelTraceEvent_Severity_name[int32(s)]
}

func prettyConnectivityState(state zpb.ChannelConnectivityState_State) string {
	return zpb.ChannelConnectivityState_State_name[int32(state)]
}

func prettyAddress(addr *zpb.Address) string {
	if ipPort := addr.GetTcpipAddress(); ipPort != nil {
		var ip net.IP = net.IP(ipPort.IpAddress)
		return fmt.Sprintf("%v:%v", ip, ipPort.Port)
	}
	// TODO: Other Address types
	return ""
}

func printChannelTraceEvents(events []*zpb.ChannelTraceEvent) {
	fmt.Fprintln(w, "Severity\tTime\tChild Ref\tDescription\t")
	for _, event := range events {
		var entry = channelTraceEventInspectEntry{
			Severity:    prettySeverity(event.Severity),
			Time:        prettyTime(event.Timestamp),
			Description: event.Description,
		}
		switch event.ChildRef.(type) {
		case *zpb.ChannelTraceEvent_SubchannelRef:
			entry.ChildRef = fmt.Sprintf("subchannel(%v)", event.GetSubchannelRef())
		case *zpb.ChannelTraceEvent_ChannelRef:
			entry.ChildRef = fmt.Sprintf("channel(%v)", event.GetChannelRef())
		}
		fmt.Fprintf(
			w, "%v\t%v\t%v\t%v\t\n",
			entry.Severity,
			entry.Time,
			entry.ChildRef,
			entry.Description,
		)
	}
	w.Flush()
}

func channelzChannelsCommandRunWithError(cmd *cobra.Command, args []string) error {
	var channels = transport.Channels()

	fmt.Fprintln(w, "Channel ID\tTarget\tState\tCalls\tResolver\tLoad-Balancing Policy\tCreated Time\t")

	for _, channel := range channels {
		var entry = channelInspectEntry{
			ID:          channel.Ref.ChannelId,
			Target:      channel.Data.Target,
			State:       prettyConnectivityState(channel.Data.State.State),
			Calls:       channel.Data.CallsStarted,
			Resolver:    "example:///",
			CreatedTime: prettyTime(channel.Data.Trace.CreationTimestamp),
		}

		for _, event := range channel.Data.Trace.Events {
			if strings.Contains(event.Description, "pick_first") {
				entry.LBPolicy = "pick_first"
			} else if strings.Contains(event.Description, "round_robin") {
				entry.LBPolicy = "round_robin"
			}
		}

		fmt.Fprintf(
			w, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t\n",
			entry.ID,
			entry.Target,
			entry.State,
			entry.Calls,
			entry.Resolver,
			entry.LBPolicy,
			entry.CreatedTime,
		)
	}

	w.Flush()
	return nil
}

var channelzChannelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "List client channels for the target application.",
	Args:  cobra.NoArgs,
	RunE:  channelzChannelsCommandRunWithError,
}

func channelzChannelCommandRunWithError(cmd *cobra.Command, args []string) error {
	var idOrTarget string = args[0]
	var selected *zpb.Channel
	var channels []*zpb.Channel = transport.Channels()

	if id, err := strconv.ParseInt(idOrTarget, 10, 64); err == nil {
		for _, channel := range channels {
			if channel.Ref.ChannelId == id {
				selected = channel
				break
			}
		}
	} else {
		for _, channel := range channels {
			if channel.Data.Target == idOrTarget {
				if selected != nil {
					return fmt.Errorf("More than one channel is connecting to target %v", idOrTarget)
				}
				selected = channel
			}
		}
	}

	if selected == nil {
		return fmt.Errorf("Cannot find channel with ID or target equal to %v", idOrTarget)
	}

	if len(selected.Data.Trace.Events) != 0 {
		printChannelTraceEvents(selected.Data.Trace.Events)
		fmt.Println("---")
	}

	fmt.Fprintln(w, "Subchannel ID\tTarget\tState\tCalls\tCreatedTime\t")

	for _, subchannelRef := range selected.SubchannelRef {
		var subchannel = transport.Subchannel(subchannelRef.SubchannelId)
		var entry = channelInspectEntry{
			ID:          subchannel.Ref.SubchannelId,
			Target:      subchannel.Data.Target,
			Calls:       subchannel.Data.CallsStarted,
			State:       prettyConnectivityState(subchannel.Data.State.State),
			CreatedTime: prettyTime(subchannel.Data.Trace.CreationTimestamp),
		}
		fmt.Fprintf(
			w, "%v\t%v\t%v\t%v\t%v\t\n",
			entry.ID,
			entry.Target,
			entry.State,
			entry.Calls,
			entry.CreatedTime,
		)
	}

	w.Flush()

	return nil
}

var channelzChannelCmd = &cobra.Command{
	Use:   "channel <channel id or URL>",
	Short: "Display channel states in human readable way.",
	Args:  cobra.ExactArgs(1),
	RunE:  channelzChannelCommandRunWithError,
}

func channelzSubchannelCommandRunWithError(cmd *cobra.Command, args []string) error {
	var idOrTarget string = args[0]
	var selected *zpb.Subchannel
	var subchannels []*zpb.Subchannel = transport.Subchannels()

	if id, err := strconv.ParseInt(idOrTarget, 10, 64); err == nil {
		for _, subchannel := range subchannels {
			if subchannel.Ref.SubchannelId == id {
				selected = subchannel
				break
			}
		}
	} else {
		for _, subchannel := range subchannels {
			if subchannel.Data.Target == idOrTarget {
				if selected != nil {
					return fmt.Errorf("More than one subchannel is connecting to target %v", idOrTarget)
				}
				selected = subchannel
			}
		}
	}

	if selected == nil {
		return fmt.Errorf("Cannot find subchannel with ID or target equal to %v", idOrTarget)
	}

	fmt.Fprintln(w, "Socket ID\tLocal->Remote\tStreams\tMessages\t")

	for _, socketRef := range selected.SocketRef {
		var socket = transport.Socket(socketRef.SocketId)
		var entry = socketInspectEntry{
			SocketID: socket.Ref.SocketId,
			Address:  fmt.Sprintf("%v->%v", prettyAddress(socket.Local), prettyAddress(socket.Remote)),
			Streams:  socket.Data.StreamsStarted,
			Messages: socket.Data.MessagesSent,
		}
		fmt.Fprintf(
			w, "%v\t%v\t%v\t%v\t\n",
			entry.SocketID,
			entry.Address,
			entry.Streams,
			entry.Messages,
		)
	}

	w.Flush()
	return nil
}

var channelzSubchannelCmd = &cobra.Command{
	Use:   "subchannel",
	Short: "Display subchannel states in human readable way.",
	Args:  cobra.ExactArgs(1),
	RunE:  channelzSubchannelCommandRunWithError,
}

var channelzCmd = &cobra.Command{
	Use:   "channelz",
	Short: "Display gRPC states in human readable way.",
	Args:  cobra.NoArgs,
}

func init() {
	channelzCmd.AddCommand(channelzChannelCmd)
	channelzCmd.AddCommand(channelzChannelsCmd)
	channelzCmd.AddCommand(channelzSubchannelCmd)
	rootCmd.AddCommand(channelzCmd)
}
