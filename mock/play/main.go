// Some command-line tools, like the `go` tool or `git`
// have many *subcommands*, each with its own set of
// flags. For example, `go build` and `go get` are two
// different subcommands of the `go` tool.
// The `flag` package lets us easily define simple
// subcommands that have their own flags.

package main

import (
	"encoding/hex"
	"fmt"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v3"
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	s := "0ac10612dd0212da020a01311ad4020a127365727665722e6578616d706c652e636f6d12bb020a013112a7020a35747970652e676f6f676c65617069732e636f6d2f656e766f792e636f6e6669672e6c697374656e65722e76332e4c697374656e657212ed019a01d5010ad2010a65747970652e676f6f676c65617069732e636f6d2f656e766f792e657874656e73696f6e732e66696c746572732e6e6574776f726b2e687474705f636f6e6e656374696f6e5f6d616e616765722e76332e48747470436f6e6e656374696f6e4d616e6167657212692a4e0a06726f7574657222440a42747970652e676f6f676c65617069732e636f6d2f656e766f792e657874656e73696f6e732e66696c746572732e687474702e726f757465722e76332e526f757465721a170a021a001211726f7574655f636f6e6669675f6e616d650a127365727665722e6578616d706c652e636f6d1a0c08b1addf810610839b94ac0230031289012286011a83010a0131126e0a3c747970652e676f6f676c65617069732e636f6d2f656e766f792e636f6e6669672e726f7574652e76332e526f757465436f6e66696775726174696f6e122e0a11726f7574655f636f6e6669675f6e616d65121912012a1a140a020a00120e0a0c636c75737465725f6e616d651a0c08b1addf81061086ccb3b1022803127b1a790a01311a740a0131125f0a33747970652e676f6f676c65617069732e636f6d2f656e766f792e636f6e6669672e636c75737465722e76332e436c757374657212281a160a021a0012106564735f736572766963655f6e616d650a0c636c75737465725f6e616d6510031a0c08b1addf810610acfdd2b602280312d50132d2011acf010a013112b9010a42747970652e676f6f676c65617069732e636f6d2f656e766f792e636f6e6669672e656e64706f696e742e76332e436c75737465724c6f616441737369676e6d656e7412730a106564735f736572766963655f6e616d65125f12140a120a100a0e12093132372e302e302e31188a551a0208030a430a1b7864735f64656661756c745f6c6f63616c6974795f726567696f6e1a096c6f63616c6974793012197864735f64656661756c745f6c6f63616c6974795f7a6f6e651a0c08b1addf810610efaef2bb022803"
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	var resp csdspb.ClientStatusResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		panic(err)
	}
	jsonbytes := protojson.Format(resp.Config[0])
	fmt.Println(string(jsonbytes))

	// fmt.Println(testdata.Path("ca.pem"))
	// // We declare a subcommand using the `NewFlagSet`
	// // function, and proceed to define new flags specific
	// // for this subcommand.
	// fooCmd := flag.NewFlagSet("foo", flag.ExitOnError)
	// fooEnable := fooCmd.Bool("enable", false, "enable")
	// fooName := fooCmd.String("name", "", "name")

	// // For a different subcommand we can define different
	// // supported flags.
	// barCmd := flag.NewFlagSet("bar", flag.ExitOnError)
	// barLevel := barCmd.Int("level", 0, "level")

	// // The subcommand is expected as the first argument
	// // to the program.
	// if len(os.Args) < 2 {
	// 	flag.Usage()
	// 	fmt.Println("expected 'foo' or 'bar' subcommands")
	// 	os.Exit(1)
	// }

	// // Check which subcommand is invoked.
	// switch os.Args[1] {

	// // For every subcommand, we parse its own flags and
	// // have access to trailing positional arguments.
	// case "foo":
	// 	fooCmd.Parse(os.Args[2:])
	// 	fmt.Println("subcommand 'foo'")
	// 	fmt.Println("  enable:", *fooEnable)
	// 	fmt.Println("  name:", *fooName)
	// 	fmt.Println("  tail:", fooCmd.Args())
	// case "bar":
	// 	barCmd.Parse(os.Args[2:])
	// 	fmt.Println("subcommand 'bar'")
	// 	fmt.Println("  level:", *barLevel)
	// 	fmt.Println("  tail:", barCmd.Args())
	// default:
	// 	fmt.Println("expected 'foo' or 'bar' subcommands")
	// 	os.Exit(1)
	// }
}
