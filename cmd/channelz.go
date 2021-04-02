package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"text/tabwriter"

	"grpcdebug/transport"

	"github.com/dustin/go-humanize"
	"github.com/golang/protobuf/ptypes"
	timestamppb "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/spf13/cobra"
	zpb "google.golang.org/grpc/channelz/grpc_channelz_v1"
)

var (
	jsonOutputFlag bool
)

// The table formater
var w = tabwriter.NewWriter(os.Stdout, 10, 0, 3, ' ', 0)

func prettyTime(ts *timestamppb.Timestamp) string {
	if timestampFlag {
		return ptypes.TimestampString(ts)
	}
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
	panic(fmt.Sprintf("Address type not supported for %s", addr))
}

func printChannelTraceEvents(events []*zpb.ChannelTraceEvent) {
	fmt.Fprintln(w, "Severity\tTime\tChild Ref\tDescription\t")
	for _, event := range events {
		var childRef string
		switch event.ChildRef.(type) {
		case *zpb.ChannelTraceEvent_SubchannelRef:
			childRef = fmt.Sprintf("subchannel(%v)", event.GetSubchannelRef())
		case *zpb.ChannelTraceEvent_ChannelRef:
			childRef = fmt.Sprintf("channel(%v)", event.GetChannelRef())
		}
		fmt.Fprintf(
			w, "%v\t%v\t%v\t%v\t\n",
			prettySeverity(event.Severity),
			prettyTime(event.Timestamp),
			childRef,
			event.Description,
		)
	}
	w.Flush()
}

func printSockets(sockets []*zpb.Socket) {
	fmt.Fprintln(w, "Socket ID\tLocal->Remote\tStreams(Started/Succeeded/Failed)\tMessages(Sent/Received)\t")
	for _, socket := range sockets {
		fmt.Fprintf(
			w, "%v\t%v\t%v/%v/%v\t%v/%v\t\n",
			socket.Ref.SocketId,
			fmt.Sprintf("%v->%v", prettyAddress(socket.Local), prettyAddress(socket.Remote)),
			socket.Data.StreamsStarted,
			socket.Data.StreamsSucceeded,
			socket.Data.StreamsFailed,
			socket.Data.MessagesSent,
			socket.Data.MessagesReceived,
		)
	}
	w.Flush()
}

func printAsJson(data interface{}) error {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(json))
	return nil
}

func channelzChannelsCommandRunWithError(cmd *cobra.Command, args []string) error {
	var channels = transport.Channels()
	// Print as JSON
	if jsonOutputFlag {
		return printAsJson(channels)
	}
	// Print as table
	fmt.Fprintln(w, "Channel ID\tTarget\tState\tCalls(Started/Succeeded/Failed)\tCreated Time\t")
	for _, channel := range channels {
		fmt.Fprintf(
			w, "%v\t%v\t%v\t%v/%v/%v\t%v\t\n",
			channel.Ref.ChannelId,
			channel.Data.Target,
			prettyConnectivityState(channel.Data.State.State),
			channel.Data.CallsStarted,
			channel.Data.CallsSucceeded,
			channel.Data.CallsFailed,
			prettyTime(channel.Data.Trace.CreationTimestamp),
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
		// Find by ID
		for _, channel := range channels {
			if channel.Ref.ChannelId == id {
				selected = channel
				break
			}
		}
	} else {
		// Find by matching target
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
	// Print as JSON
	if jsonOutputFlag {
		return printAsJson(selected)
	}
	// Print as table
	// Print Channel information
	fmt.Fprintf(w, "Channel ID:\t%v\t\n", selected.Ref.ChannelId)
	fmt.Fprintf(w, "Target:\t%v\t\n", selected.Data.Target)
	fmt.Fprintf(w, "State:\t%v\t\n", prettyConnectivityState(selected.Data.State.State))
	fmt.Fprintf(w, "Calls Started:\t%v\t\n", selected.Data.CallsStarted)
	fmt.Fprintf(w, "Calls Succeeded:\t%v\t\n", selected.Data.CallsSucceeded)
	fmt.Fprintf(w, "Calls Failed:\t%v\t\n", selected.Data.CallsFailed)
	fmt.Fprintf(w, "Created Time:\t%v\t\n", prettyTime(selected.Data.Trace.CreationTimestamp))
	w.Flush()
	// Print Subchannel list
	if len(selected.SubchannelRef) > 0 {
		fmt.Println("---")
		fmt.Fprintln(w, "Subchannel ID\tTarget\tState\tCalls(Started/Succeeded/Failed)\tCreatedTime\t")
		for _, subchannelRef := range selected.SubchannelRef {
			var subchannel = transport.Subchannel(subchannelRef.SubchannelId)
			fmt.Fprintf(
				w, "%v\t%v\t%v\t%v/%v/%v\t%v\t\n",
				subchannel.Ref.SubchannelId,
				subchannel.Data.Target,
				prettyConnectivityState(subchannel.Data.State.State),
				subchannel.Data.CallsStarted,
				subchannel.Data.CallsSucceeded,
				subchannel.Data.CallsFailed,
				prettyTime(subchannel.Data.Trace.CreationTimestamp),
			)
		}
		w.Flush()
	}
	// Print channel trace events
	if len(selected.Data.Trace.Events) != 0 {
		fmt.Println("---")
		printChannelTraceEvents(selected.Data.Trace.Events)
	}
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
	// Print as JSON
	if jsonOutputFlag {
		return printAsJson(selected)
	}
	// Print as table
	// Print Subchannel information
	fmt.Fprintf(w, "Subchannel ID:\t%v\t\n", selected.Ref.SubchannelId)
	fmt.Fprintf(w, "Target:\t%v\t\n", selected.Data.Target)
	fmt.Fprintf(w, "State:\t%v\t\n", prettyConnectivityState(selected.Data.State.State))
	fmt.Fprintf(w, "Calls Started:\t%v\t\n", selected.Data.CallsStarted)
	fmt.Fprintf(w, "Calls Succeeded:\t%v\t\n", selected.Data.CallsSucceeded)
	fmt.Fprintf(w, "Calls Failed:\t%v\t\n", selected.Data.CallsFailed)
	fmt.Fprintf(w, "Created Time:\t%v\t\n", prettyTime(selected.Data.Trace.CreationTimestamp))
	w.Flush()
	if len(selected.SocketRef) > 0 {
		// Print socket list
		fmt.Println("---")
		var sockets []*zpb.Socket
		for _, socketRef := range selected.SocketRef {
			sockets = append(sockets, transport.Socket(socketRef.SocketId))
		}
		printSockets(sockets)
	}
	return nil
}

var channelzSubchannelCmd = &cobra.Command{
	Use:   "subchannel",
	Short: "Display subchannel states in human readable way.",
	Args:  cobra.ExactArgs(1),
	RunE:  channelzSubchannelCommandRunWithError,
}

func channelzSocketCommandRunWithError(cmd *cobra.Command, args []string) error {
	socketId, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid socket ID %v", socketId)
	}
	selected := transport.Socket(socketId)
	if selected == nil {
		return fmt.Errorf("Cannot find socket with ID %v", socketId)
	}
	// Print as JSON
	if jsonOutputFlag {
		return printAsJson(selected)
	}
	// Print as table
	// Print Socket information
	fmt.Fprintf(w, "Socket ID:\t%v\t\n", selected.Ref.SocketId)
	fmt.Fprintf(w, "Address:\t%v\t\n", fmt.Sprintf("%v->%v", prettyAddress(selected.Local), prettyAddress(selected.Remote)))
	fmt.Fprintf(w, "Streams Started:\t%v\t\n", selected.Data.StreamsStarted)
	fmt.Fprintf(w, "Streams Succeeded:\t%v\t\n", selected.Data.StreamsSucceeded)
	fmt.Fprintf(w, "Streams Failed:\t%v\t\n", selected.Data.StreamsFailed)
	fmt.Fprintf(w, "Messages Sent:\t%v\t\n", selected.Data.MessagesSent)
	fmt.Fprintf(w, "Messages Received:\t%v\t\n", selected.Data.MessagesReceived)
	fmt.Fprintf(w, "Keep Alives Sent:\t%v\t\n", selected.Data.KeepAlivesSent)
	fmt.Fprintf(w, "Last Local Stream Created:\t%v\t\n", prettyTime(selected.Data.LastLocalStreamCreatedTimestamp))
	fmt.Fprintf(w, "Last Remote Stream Created:\t%v\t\n", prettyTime(selected.Data.LastRemoteStreamCreatedTimestamp))
	fmt.Fprintf(w, "Last Message Sent Created:\t%v\t\n", prettyTime(selected.Data.LastMessageSentTimestamp))
	fmt.Fprintf(w, "Last Message Received Created:\t%v\t\n", prettyTime(selected.Data.LastMessageReceivedTimestamp))
	fmt.Fprintf(w, "Local Flow Control Window:\t%v\t\n", selected.Data.LocalFlowControlWindow.Value)
	fmt.Fprintf(w, "Remote Flow Control Window:\t%v\t\n", selected.Data.RemoteFlowControlWindow.Value)
	w.Flush()
	if len(selected.Data.Option) > 0 {
		fmt.Println("---")
		fmt.Fprintln(w, "Socket Options Name\tValue\t")
		for _, option := range selected.Data.Option {
			if option.Value != "" {
				// Prefer human readable value than the Any proto
				fmt.Fprintf(w, "%v\t%v\t\n", option.Name, option.Value)
			} else {
				fmt.Fprintf(w, "%v\t%v\t\n", option.Name, option.Additional)
			}
		}
		w.Flush()
	}
	// Print security information
	if security := selected.GetSecurity(); security != nil {
		fmt.Println("---")
		switch x := security.Model.(type) {
		case *zpb.Security_Tls_:
			fmt.Fprintf(w, "Security Model:\t%v\t\n", "TLS")
			switch y := security.GetTls().CipherSuite.(type) {
			case *zpb.Security_Tls_StandardName:
				fmt.Fprintf(w, "Standard Name:\t%v\t\n", security.GetTls().GetStandardName())
			case *zpb.Security_Tls_OtherName:
				fmt.Fprintf(w, "Other Name:\t%v\t\n", security.GetTls().GetOtherName())
			default:
				return fmt.Errorf("Unexpected Cipher suite name type %T", y)
			}
			// fmt.Fprintf(w, "Local Certificate:\t%v\t\n", security.GetTls().LocalCertificate)
			// fmt.Fprintf(w, "Remote Certificate:\t%v\t\n", security.GetTls().RemoteCertificate)
		case *zpb.Security_Other:
			fmt.Fprintf(w, "Security Model:\t%v\t\n", "Other")
			fmt.Fprintf(w, "Name:\t%v\t\n", security.GetOther().Name)
			// fmt.Fprintf(w, "Value:\t%v\t\n", security.GetOther().Value)
		default:
			return fmt.Errorf("Unexpected security model type %T", x)
		}
		w.Flush()
	}
	return nil
}

var channelzSocketCmd = &cobra.Command{
	Use:   "socket",
	Short: "Display socket states in human readable way.",
	Args:  cobra.ExactArgs(1),
	RunE:  channelzSocketCommandRunWithError,
}

func channelzServersCommandRunWithError(cmd *cobra.Command, args []string) error {
	var servers = transport.Servers()
	// Print as JSON
	if jsonOutputFlag {
		return printAsJson(servers)
	}
	// Print as table
	fmt.Fprintln(w, "Server ID\tListenAddresses\tCallsStarted\tCallsSucceeded\tCallsFailed\tCreated Time\t")
	for _, server := range servers {
		var listenAddresses []string
		for _, socketRef := range server.ListenSocket {
			socket := transport.Socket(socketRef.SocketId)
			listenAddresses = append(listenAddresses, prettyAddress(socket.Local))
		}
		fmt.Fprintf(
			w, "%v\t%v\t%v\t%v\t%v\t%v\t\n",
			server.Ref.ServerId,
			listenAddresses,
			server.Data.CallsStarted,
			server.Data.CallsSucceeded,
			server.Data.CallsFailed,
			prettyTime(server.Data.LastCallStartedTimestamp),
		)
	}
	w.Flush()
	return nil
}

var channelzServersCmd = &cobra.Command{
	Use:   "servers",
	Short: "List servers in human readable way.",
	Args:  cobra.NoArgs,
	RunE:  channelzServersCommandRunWithError,
}

func channelzServerCommandRunWithError(cmd *cobra.Command, args []string) error {
	var servers = transport.Servers()
	var selected *zpb.Server
	serverId, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid server ID %v", serverId)
	}
	for _, server := range servers {
		if server.Ref.ServerId == serverId {
			selected = server
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("Cannot find server with ID %v", serverId)
	}
	// Print as JSON
	if jsonOutputFlag {
		return printAsJson(servers)
	}
	// Print as table
	var listenAddresses []string
	for _, socketRef := range selected.ListenSocket {
		socket := transport.Socket(socketRef.SocketId)
		listenAddresses = append(listenAddresses, prettyAddress(socket.Local))
	}
	fmt.Fprintf(w, "Server Id:\t%v\t\n", selected.Ref.ServerId)
	fmt.Fprintf(w, "Listen Addresses:\t%v\t\n", listenAddresses)
	fmt.Fprintf(w, "Calls Started:\t%v\t\n", selected.Data.CallsStarted)
	fmt.Fprintf(w, "Calls Succeeded:\t%v\t\n", selected.Data.CallsSucceeded)
	fmt.Fprintf(w, "Calls Failed:\t%v\t\n", selected.Data.CallsFailed)
	fmt.Fprintf(w, "Created Time:\t%v\t\n", prettyTime(selected.Data.LastCallStartedTimestamp))
	w.Flush()
	if sockets := transport.ServerSocket(selected.Ref.ServerId); len(sockets) > 0 {
		// Print socket list
		fmt.Println("---")
		printSockets(sockets)
	}
	return nil
}

var channelzServerCmd = &cobra.Command{
	Use:   "server <id>",
	Short: "Display server state in human readable way.",
	Args:  cobra.ExactArgs(1),
	RunE:  channelzServerCommandRunWithError,
}

var channelzCmd = &cobra.Command{
	Use:   "channelz",
	Short: "Display gRPC states in human readable way.",
	Args:  cobra.NoArgs,
}

func init() {
	channelzCmd.PersistentFlags().BoolVarP(&jsonOutputFlag, "json", "o", false, "Whether to print the result as JSON")
	channelzCmd.AddCommand(channelzChannelCmd)
	channelzCmd.AddCommand(channelzChannelsCmd)
	channelzCmd.AddCommand(channelzSubchannelCmd)
	channelzCmd.AddCommand(channelzSocketCmd)
	channelzCmd.AddCommand(channelzServersCmd)
	channelzCmd.AddCommand(channelzServerCmd)
	rootCmd.AddCommand(channelzCmd)
}
