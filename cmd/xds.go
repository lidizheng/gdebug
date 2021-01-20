package cmd

import (
	"fmt"

	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v3"
	"github.com/spf13/cobra"
)

var configDump = `{
	"xdsConfig": [{
		"clientStatus": 2,
		"listenerConfig": {
			"dynamicListeners": [{
				"activeState": {
					"lastUpdated": "2021-01-20T19:46:14.720363332Z",
					"listener": {
						"@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
						"apiListener": {
							"apiListener": {
								"@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
								"routeConfig": {
									"name": "route_config_name",
									"virtualHosts": [{
										"domains": ["*"],
										"routes": [{
											"match": {
												"prefix": ""
											},
											"route": {
												"cluster": "cluster_name"
											}
										}]
									}]
								}
							}
						},
						"name": "x.test.youtube.com"
					},
					"versionInfo": "1"
				},
				"name": "x.test.youtube.com"
			}],
			"versionInfo": "1"
		}
	}, {
		"clientStatus": 0,
		"routeConfig": {
			"dynamicRouteConfigs": []
		}
	}, {
		"clientStatus": 2,
		"clusterConfig": {
			"dynamicActiveClusters": [{
				"cluster": {
					"@type": "type.googleapis.com/envoy.config.cluster.v3.Cluster",
					"edsClusterConfig": {
						"edsConfig": {
							"ads": {}
						},
						"serviceName": "eds_service_name"
					},
					"name": "cluster_name",
					"type": "EDS"
				},
				"lastUpdated": "2021-01-20T19:46:14.731363449Z",
				"versionInfo": "1"
			}],
			"versionInfo": "1"
		}
	}, {
		"clientStatus": 2,
		"endpointConfig": {
			"dynamicEndpointConfigs": [{
				"endpointConfig": {
					"@type": "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment",
					"clusterName": "eds_service_name",
					"endpoints": [{
						"lbEndpoints": [{
							"endpoint": {
								"address": {
									"socketAddress": {
										"address": "127.0.0.1",
										"portValue": 10001
									}
								}
							}
						}, {
							"endpoint": {
								"address": {
									"socketAddress": {
										"address": "127.0.0.1",
										"portValue": 10002
									}
								}
							}
						}, {
							"endpoint": {
								"address": {
									"socketAddress": {
										"address": "127.0.0.1",
										"portValue": 10003
									}
								}
							}
						}],
						"loadBalancingWeight": 3,
						"locality": {
							"region": "xds_default_locality_region",
							"subZone": "locality0",
							"zone": "xds_default_locality_zone"
						}
					}]
				},
				"lastUpdated": "2021-01-20T19:46:14.741363465Z",
				"versionInfo": "1"
			}],
			"versionInfo": "1"
		}
	}]
}`

var configListener = `{
	"dynamicListeners": [{
		"activeState": {
			"lastUpdated": "2021-01-20T19:46:14.720363332Z",
			"listener": {
				"@type": "type.googleapis.com/envoy.config.listener.v3.Listener",
				"apiListener": {
					"apiListener": {
						"@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
						"routeConfig": {
							"name": "route_config_name",
							"virtualHosts": [{
								"domains": ["*"],
								"routes": [{
									"match": {
										"prefix": ""
									},
									"route": {
										"cluster": "cluster_name"
									}
								}]
							}]
						}
					}
				},
				"name": "x.test.youtube.com"
			},
			"versionInfo": "1"
		},
		"name": "x.test.youtube.com"
	}],
	"versionInfo": "1"
}`

func xdsConfigCommandRunWithError(cmd *cobra.Command, args []string) error {
	// clientStatus := transport.FetchClientStatus()
	// fmt.Print(protojson.Format(clientStatus))
	if len(args) > 0 && args[0] == "listeners" {
		fmt.Print(configListener)
	} else {
		fmt.Print(configDump)
	}
	return nil
}

var xdsConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Dump the operating xDS configs.",
	RunE:  xdsConfigCommandRunWithError,
}

type xdsStatusEntry struct {
	Name    string
	Status  string
	Version string
}

func prettyClientConfigStatus(s csdspb.ClientConfigStatus) string {
	return csdspb.ClientConfigStatus_name[int32(s)]
}

func xdsStatusCommandRunWithError(cmd *cobra.Command, args []string) error {
	// clientStatus := transport.FetchClientStatus()

	fmt.Fprintln(w, "Name\tStatus\tVersion\t")

	// config := clientStatus.Config[0]
	// for _, xdsConfig := range config.XdsConfig {
	// 	var entry = xdsStatusEntry{
	// 		Status: prettyClientConfigStatus(xdsConfig.ClientStatus),
	// 	}

	// 	switch x := xdsConfig.PerXdsConfig.(type) {
	// 	case *adminpb.ListenersConfigDump:
	// 		entry.Name = "LDS"
	// 		entry.Version = xdsConfig.PerXdsConfig.VersionInfo
	// 	case *adminpb.ClustersConfigDump:
	// 		entry.Name = "CDS"
	// 	case *adminpb.RoutesConfigDump:
	// 		entry.Name = "RDS"
	// 	case *adminpb.ScopedRoutesConfigDump:
	// 		log.Panic("Not implemented")
	// 	case *adminpb.EndpointsConfigDump:
	// 		entry.Name = "EDS"
	// 	default:
	// 		log.Fatalf("failed to read per_xds_config value")
	// 	}

	// 	fmt.Fprintf(
	// 		w, "%v\t%v\t%v\t\n",
	// 		entry.Name,
	// 		entry.Status,
	// 		entry.Version,
	// 	)
	// }

	fmt.Fprintf(
		w, "%v\t%v\t%v\t\n%v\t%v\t%v\t\n%v\t%v\t%v\t\n%v\t%v\t%v\t\n",
		"LDS", "CLIENT_ACKED", "1594852577637078317",
		"RDS", "CLIENT_ACKED", "1595268153502909101",
		"CDS", "CLIENT_REQUESTED", "N/A",
		"EDS", "CLIENT_ACKED", "1595644593548779066",
	)

	w.Flush()
	return nil
}

var xdsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print the config synchronization status.",
	RunE:  xdsStatusCommandRunWithError,
}

var xdsCmd = &cobra.Command{
	Use:   "xds",
	Short: "Fetch xDS related information.",
}

func init() {
	xdsCmd.AddCommand(xdsConfigCmd)
	xdsCmd.AddCommand(xdsStatusCmd)
	rootCmd.AddCommand(xdsCmd)
}
