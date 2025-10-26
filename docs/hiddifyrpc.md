# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [v2/config/route_rule.proto](#v2_config_route_rule-proto)
    - [RouteRule](#config-RouteRule)
    - [Rule](#config-Rule)
  
    - [Network](#config-Network)
    - [Outbound](#config-Outbound)
    - [Protocol](#config-Protocol)
  
- [v2/hcore/hcore.proto](#v2_hcore_hcore-proto)
    - [ChangeHiddifySettingsRequest](#hcore-ChangeHiddifySettingsRequest)
    - [CoreInfoResponse](#hcore-CoreInfoResponse)
    - [GenerateConfigRequest](#hcore-GenerateConfigRequest)
    - [GenerateConfigResponse](#hcore-GenerateConfigResponse)
    - [GenerateWarpConfigRequest](#hcore-GenerateWarpConfigRequest)
    - [IpInfo](#hcore-IpInfo)
    - [LogMessage](#hcore-LogMessage)
    - [OutboundGroup](#hcore-OutboundGroup)
    - [OutboundGroupList](#hcore-OutboundGroupList)
    - [OutboundInfo](#hcore-OutboundInfo)
    - [ParseRequest](#hcore-ParseRequest)
    - [ParseResponse](#hcore-ParseResponse)
    - [PauseRequest](#hcore-PauseRequest)
    - [SelectOutboundRequest](#hcore-SelectOutboundRequest)
    - [SetSystemProxyEnabledRequest](#hcore-SetSystemProxyEnabledRequest)
    - [SetupRequest](#hcore-SetupRequest)
    - [StartRequest](#hcore-StartRequest)
    - [StopRequest](#hcore-StopRequest)
    - [SystemInfo](#hcore-SystemInfo)
    - [SystemProxyStatus](#hcore-SystemProxyStatus)
    - [UrlTestRequest](#hcore-UrlTestRequest)
    - [WarpAccount](#hcore-WarpAccount)
    - [WarpGenerationResponse](#hcore-WarpGenerationResponse)
    - [WarpWireguardConfig](#hcore-WarpWireguardConfig)
  
    - [CoreStates](#hcore-CoreStates)
    - [LogLevel](#hcore-LogLevel)
    - [LogType](#hcore-LogType)
    - [MessageType](#hcore-MessageType)
    - [SetupMode](#hcore-SetupMode)
  
- [v2/hcore/hcore_service.proto](#v2_hcore_hcore_service-proto)
    - [Core](#hcore-Core)
  
- [v2/hcore/tunnelservice/tunnel.proto](#v2_hcore_tunnelservice_tunnel-proto)
    - [TunnelResponse](#tunnelservice-TunnelResponse)
    - [TunnelStartRequest](#tunnelservice-TunnelStartRequest)
  
- [v2/hcore/tunnelservice/tunnel_service.proto](#v2_hcore_tunnelservice_tunnel_service-proto)
    - [TunnelService](#tunnelservice-TunnelService)
  
- [v2/hiddifyoptions/hiddify_options.proto](#v2_hiddifyoptions_hiddify_options-proto)
    - [DNSOptions](#hiddifyoptions-DNSOptions)
    - [HiddifyOptions](#hiddifyoptions-HiddifyOptions)
    - [InboundOptions](#hiddifyoptions-InboundOptions)
    - [IntRange](#hiddifyoptions-IntRange)
    - [MuxOptions](#hiddifyoptions-MuxOptions)
    - [RouteOptions](#hiddifyoptions-RouteOptions)
    - [Rule](#hiddifyoptions-Rule)
    - [TLSTricks](#hiddifyoptions-TLSTricks)
    - [URLTestOptions](#hiddifyoptions-URLTestOptions)
    - [WarpAccount](#hiddifyoptions-WarpAccount)
    - [WarpOptions](#hiddifyoptions-WarpOptions)
    - [WarpWireguardConfig](#hiddifyoptions-WarpWireguardConfig)
  
    - [DomainStrategy](#hiddifyoptions-DomainStrategy)
  
- [v2/profile/profile_service.proto](#v2_profile_profile_service-proto)
    - [AddProfileRequest](#profile-AddProfileRequest)
    - [MultiProfilesResponse](#profile-MultiProfilesResponse)
    - [ProfileRequest](#profile-ProfileRequest)
    - [ProfileResponse](#profile-ProfileResponse)
  
    - [ProfileService](#profile-ProfileService)
  
- [v2/profile/profile.proto](#v2_profile_profile-proto)
    - [ProfileEntity](#profile-ProfileEntity)
    - [ProfileOptions](#profile-ProfileOptions)
    - [SubscriptionInfo](#profile-SubscriptionInfo)
  
- [v2/hcommon/common.proto](#v2_hcommon_common-proto)
    - [Empty](#hcommon-Empty)
    - [Response](#hcommon-Response)
  
    - [ResponseCode](#hcommon-ResponseCode)
  
- [v2/hello/hello_service.proto](#v2_hello_hello_service-proto)
    - [Hello](#hello-Hello)
  
- [v2/hello/hello.proto](#v2_hello_hello-proto)
    - [HelloRequest](#hello-HelloRequest)
    - [HelloResponse](#hello-HelloResponse)
  
- [extension/extension_service.proto](#extension_extension_service-proto)
    - [ExtensionHostService](#extension-ExtensionHostService)
  
- [extension/extension.proto](#extension_extension-proto)
    - [EditExtensionRequest](#extension-EditExtensionRequest)
    - [ExtensionActionResult](#extension-ExtensionActionResult)
    - [ExtensionList](#extension-ExtensionList)
    - [ExtensionMsg](#extension-ExtensionMsg)
    - [ExtensionRequest](#extension-ExtensionRequest)
    - [ExtensionRequest.DataEntry](#extension-ExtensionRequest-DataEntry)
    - [ExtensionResponse](#extension-ExtensionResponse)
    - [SendExtensionDataRequest](#extension-SendExtensionDataRequest)
    - [SendExtensionDataRequest.DataEntry](#extension-SendExtensionDataRequest-DataEntry)
  
    - [ExtensionResponseType](#extension-ExtensionResponseType)
  
- [Scalar Value Types](#scalar-value-types)



<a name="v2_config_route_rule-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/config/route_rule.proto



<a name="config-RouteRule"></a>

### RouteRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [Rule](#config-Rule) | repeated |  |






<a name="config-Rule"></a>

### Rule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| list_order | [uint32](#uint32) |  |  |
| enabled | [bool](#bool) |  |  |
| name | [string](#string) |  |  |
| outbound | [Outbound](#config-Outbound) |  |  |
| rule_sets | [string](#string) | repeated |  |
| package_names | [string](#string) | repeated |  |
| process_names | [string](#string) | repeated |  |
| process_paths | [string](#string) | repeated |  |
| network | [Network](#config-Network) |  |  |
| port_ranges | [string](#string) | repeated |  |
| source_port_ranges | [string](#string) | repeated |  |
| protocols | [Protocol](#config-Protocol) | repeated |  |
| ip_cidrs | [string](#string) | repeated |  |
| source_ip_cidrs | [string](#string) | repeated |  |
| domains | [string](#string) | repeated |  |
| domain_suffixes | [string](#string) | repeated |  |
| domain_keywords | [string](#string) | repeated |  |
| domain_regexes | [string](#string) | repeated |  |





 


<a name="config-Network"></a>

### Network


| Name | Number | Description |
| ---- | ------ | ----------- |
| all | 0 |  |
| tcp | 1 |  |
| udp | 2 |  |



<a name="config-Outbound"></a>

### Outbound


| Name | Number | Description |
| ---- | ------ | ----------- |
| proxy | 0 |  |
| direct | 1 |  |
| direct_with_fragment | 2 |  |
| block | 3 |  |



<a name="config-Protocol"></a>

### Protocol


| Name | Number | Description |
| ---- | ------ | ----------- |
| tls | 0 |  |
| http | 1 |  |
| quic | 2 |  |
| stun | 3 |  |
| dns | 4 |  |
| bittorrent | 5 |  |


 

 

 



<a name="v2_hcore_hcore-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/hcore/hcore.proto



<a name="hcore-ChangeHiddifySettingsRequest"></a>

### ChangeHiddifySettingsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hiddify_settings_json | [string](#string) |  |  |






<a name="hcore-CoreInfoResponse"></a>

### CoreInfoResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| core_state | [CoreStates](#hcore-CoreStates) |  |  |
| message_type | [MessageType](#hcore-MessageType) |  |  |
| message | [string](#string) |  |  |






<a name="hcore-GenerateConfigRequest"></a>

### GenerateConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  |  |
| temp_path | [string](#string) |  |  |
| debug | [bool](#bool) |  |  |






<a name="hcore-GenerateConfigResponse"></a>

### GenerateConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config_content | [string](#string) |  |  |






<a name="hcore-GenerateWarpConfigRequest"></a>

### GenerateWarpConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| license_key | [string](#string) |  |  |
| account_id | [string](#string) |  |  |
| access_token | [string](#string) |  |  |






<a name="hcore-IpInfo"></a>

### IpInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip | [string](#string) |  | The IP address. |
| country_code | [string](#string) |  | The country code. |
| region | [string](#string) |  | The region (optional). |
| city | [string](#string) |  | The city (optional). |
| asn | [int32](#int32) |  | The Autonomous System Number (optional). |
| org | [string](#string) |  | The organization (optional). |
| latitude | [double](#double) |  | The latitude (optional). |
| longitude | [double](#double) |  | The longitude (optional). |
| postal_code | [string](#string) |  | The postal code (optional). |






<a name="hcore-LogMessage"></a>

### LogMessage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [LogLevel](#hcore-LogLevel) |  |  |
| type | [LogType](#hcore-LogType) |  |  |
| message | [string](#string) |  |  |
| time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="hcore-OutboundGroup"></a>

### OutboundGroup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tag | [string](#string) |  |  |
| type | [string](#string) |  |  |
| selected | [OutboundInfo](#hcore-OutboundInfo) |  |  |
| selectable | [bool](#bool) |  |  |
| Is_expand | [bool](#bool) |  |  |
| items | [OutboundInfo](#hcore-OutboundInfo) | repeated |  |






<a name="hcore-OutboundGroupList"></a>

### OutboundGroupList



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [OutboundGroup](#hcore-OutboundGroup) | repeated |  |






<a name="hcore-OutboundInfo"></a>

### OutboundInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tag | [string](#string) |  |  |
| type | [string](#string) |  |  |
| url_test_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| url_test_delay | [int32](#int32) |  |  |
| ipinfo | [IpInfo](#hcore-IpInfo) | optional |  |
| is_selected | [bool](#bool) |  |  |
| is_group | [bool](#bool) |  |  |
| group_selected_outbound | [OutboundInfo](#hcore-OutboundInfo) | optional |  |
| is_secure | [bool](#bool) |  |  |
| is_visible | [bool](#bool) |  |  |
| port | [uint32](#uint32) |  |  |
| host | [string](#string) |  |  |
| tag_display | [string](#string) |  |  |






<a name="hcore-ParseRequest"></a>

### ParseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [string](#string) |  |  |
| config_path | [string](#string) |  |  |
| temp_path | [string](#string) |  |  |
| debug | [bool](#bool) |  |  |






<a name="hcore-ParseResponse"></a>

### ParseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| response_code | [hcommon.ResponseCode](#hcommon-ResponseCode) |  |  |
| content | [string](#string) |  |  |
| message | [string](#string) |  |  |






<a name="hcore-PauseRequest"></a>

### PauseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mode | [SetupMode](#hcore-SetupMode) |  |  |






<a name="hcore-SelectOutboundRequest"></a>

### SelectOutboundRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group_tag | [string](#string) |  |  |
| outbound_tag | [string](#string) |  |  |






<a name="hcore-SetSystemProxyEnabledRequest"></a>

### SetSystemProxyEnabledRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| is_enabled | [bool](#bool) |  |  |






<a name="hcore-SetupRequest"></a>

### SetupRequest
Define the message equivalent of SetupParameters


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| base_path | [string](#string) |  |  |
| working_dir | [string](#string) |  |  |
| temp_dir | [string](#string) |  |  |
| flutter_status_port | [int64](#int64) |  |  |
| listen | [string](#string) |  |  |
| secret | [string](#string) |  |  |
| debug | [bool](#bool) |  |  |
| mode | [SetupMode](#hcore-SetupMode) |  |  |






<a name="hcore-StartRequest"></a>

### StartRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config_path | [string](#string) |  |  |
| config_content | [string](#string) |  | Optional if configPath is not provided. |
| disable_memory_limit | [bool](#bool) |  |  |
| delay_start | [bool](#bool) |  |  |
| enable_old_command_server | [bool](#bool) |  |  |
| enable_raw_config | [bool](#bool) |  |  |
| config_name | [string](#string) |  |  |






<a name="hcore-StopRequest"></a>

### StopRequest







<a name="hcore-SystemInfo"></a>

### SystemInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| memory | [int64](#int64) |  |  |
| goroutines | [int32](#int32) |  |  |
| connections_in | [int32](#int32) |  |  |
| connections_out | [int32](#int32) |  |  |
| traffic_available | [bool](#bool) |  |  |
| uplink | [int64](#int64) |  |  |
| downlink | [int64](#int64) |  |  |
| uplink_total | [int64](#int64) |  |  |
| downlink_total | [int64](#int64) |  |  |
| current_outbound | [string](#string) |  |  |
| current_profile | [string](#string) |  |  |






<a name="hcore-SystemProxyStatus"></a>

### SystemProxyStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| available | [bool](#bool) |  |  |
| enabled | [bool](#bool) |  |  |






<a name="hcore-UrlTestRequest"></a>

### UrlTestRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group_tag | [string](#string) |  |  |






<a name="hcore-WarpAccount"></a>

### WarpAccount



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account_id | [string](#string) |  |  |
| access_token | [string](#string) |  |  |






<a name="hcore-WarpGenerationResponse"></a>

### WarpGenerationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account | [WarpAccount](#hcore-WarpAccount) |  |  |
| log | [string](#string) |  |  |
| config | [WarpWireguardConfig](#hcore-WarpWireguardConfig) |  |  |






<a name="hcore-WarpWireguardConfig"></a>

### WarpWireguardConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| private_key | [string](#string) |  |  |
| local_address_ipv4 | [string](#string) |  |  |
| local_address_ipv6 | [string](#string) |  |  |
| peer_public_key | [string](#string) |  |  |
| client_id | [string](#string) |  |  |





 


<a name="hcore-CoreStates"></a>

### CoreStates


| Name | Number | Description |
| ---- | ------ | ----------- |
| STOPPED | 0 |  |
| STARTING | 1 |  |
| STARTED | 2 |  |
| STOPPING | 3 |  |



<a name="hcore-LogLevel"></a>

### LogLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| DEBUG | 0 |  |
| INFO | 1 |  |
| WARNING | 2 |  |
| ERROR | 3 |  |
| FATAL | 4 |  |



<a name="hcore-LogType"></a>

### LogType


| Name | Number | Description |
| ---- | ------ | ----------- |
| CORE | 0 |  |
| SERVICE | 1 |  |
| CONFIG | 2 |  |



<a name="hcore-MessageType"></a>

### MessageType


| Name | Number | Description |
| ---- | ------ | ----------- |
| EMPTY | 0 |  |
| EMPTY_CONFIGURATION | 1 |  |
| START_COMMAND_SERVER | 2 |  |
| CREATE_SERVICE | 3 |  |
| START_SERVICE | 4 |  |
| UNEXPECTED_ERROR | 5 |  |
| ALREADY_STARTED | 6 |  |
| ALREADY_STOPPED | 7 |  |
| INSTANCE_NOT_FOUND | 8 |  |
| INSTANCE_NOT_STOPPED | 9 |  |
| INSTANCE_NOT_STARTED | 10 |  |
| ERROR_BUILDING_CONFIG | 11 |  |
| ERROR_PARSING_CONFIG | 12 |  |
| ERROR_READING_CONFIG | 13 |  |
| ERROR_EXTENSION | 14 |  |



<a name="hcore-SetupMode"></a>

### SetupMode


| Name | Number | Description |
| ---- | ------ | ----------- |
| OLD | 0 |  |
| GRPC_NORMAL | 1 |  |
| GRPC_BACKGROUND | 2 |  |
| GRPC_NORMAL_INSECURE | 3 |  |
| GRPC_BACKGROUND_INSECURE | 4 |  |


 

 

 



<a name="v2_hcore_hcore_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/hcore/hcore_service.proto


 

 

 


<a name="hcore-Core"></a>

### Core


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Start | [StartRequest](#hcore-StartRequest) | [CoreInfoResponse](#hcore-CoreInfoResponse) |  |
| CoreInfoListener | [.hcommon.Empty](#hcommon-Empty) | [CoreInfoResponse](#hcore-CoreInfoResponse) stream |  |
| OutboundsInfo | [.hcommon.Empty](#hcommon-Empty) | [OutboundGroupList](#hcore-OutboundGroupList) stream |  |
| MainOutboundsInfo | [.hcommon.Empty](#hcommon-Empty) | [OutboundGroupList](#hcore-OutboundGroupList) stream |  |
| GetSystemInfo | [.hcommon.Empty](#hcommon-Empty) | [SystemInfo](#hcore-SystemInfo) stream |  |
| Setup | [SetupRequest](#hcore-SetupRequest) | [.hcommon.Response](#hcommon-Response) |  |
| Parse | [ParseRequest](#hcore-ParseRequest) | [ParseResponse](#hcore-ParseResponse) |  |
| ChangeHiddifySettings | [ChangeHiddifySettingsRequest](#hcore-ChangeHiddifySettingsRequest) | [CoreInfoResponse](#hcore-CoreInfoResponse) |  |
| StartService | [StartRequest](#hcore-StartRequest) | [CoreInfoResponse](#hcore-CoreInfoResponse) | rpc GenerateConfig (GenerateConfigRequest) returns (GenerateConfigResponse); |
| Stop | [.hcommon.Empty](#hcommon-Empty) | [CoreInfoResponse](#hcore-CoreInfoResponse) |  |
| Restart | [StartRequest](#hcore-StartRequest) | [CoreInfoResponse](#hcore-CoreInfoResponse) |  |
| SelectOutbound | [SelectOutboundRequest](#hcore-SelectOutboundRequest) | [.hcommon.Response](#hcommon-Response) |  |
| UrlTest | [UrlTestRequest](#hcore-UrlTestRequest) | [.hcommon.Response](#hcommon-Response) |  |
| GenerateWarpConfig | [GenerateWarpConfigRequest](#hcore-GenerateWarpConfigRequest) | [WarpGenerationResponse](#hcore-WarpGenerationResponse) |  |
| GetSystemProxyStatus | [.hcommon.Empty](#hcommon-Empty) | [SystemProxyStatus](#hcore-SystemProxyStatus) |  |
| SetSystemProxyEnabled | [SetSystemProxyEnabledRequest](#hcore-SetSystemProxyEnabledRequest) | [.hcommon.Response](#hcommon-Response) |  |
| LogListener | [.hcommon.Empty](#hcommon-Empty) | [LogMessage](#hcore-LogMessage) stream |  |
| Pause | [PauseRequest](#hcore-PauseRequest) | [.hcommon.Empty](#hcommon-Empty) |  |

 



<a name="v2_hcore_tunnelservice_tunnel-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/hcore/tunnelservice/tunnel.proto



<a name="tunnelservice-TunnelResponse"></a>

### TunnelResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message | [string](#string) |  |  |






<a name="tunnelservice-TunnelStartRequest"></a>

### TunnelStartRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ipv6 | [bool](#bool) |  |  |
| server_port | [int32](#int32) |  |  |
| server_username | [string](#string) |  |  |
| server_password | [string](#string) |  |  |
| strict_route | [bool](#bool) |  |  |
| endpoint_independent_nat | [bool](#bool) |  |  |
| stack | [string](#string) |  |  |





 

 

 

 



<a name="v2_hcore_tunnelservice_tunnel_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/hcore/tunnelservice/tunnel_service.proto


 

 

 


<a name="tunnelservice-TunnelService"></a>

### TunnelService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Start | [TunnelStartRequest](#tunnelservice-TunnelStartRequest) | [TunnelResponse](#tunnelservice-TunnelResponse) |  |
| Stop | [.hcommon.Empty](#hcommon-Empty) | [TunnelResponse](#tunnelservice-TunnelResponse) |  |
| Status | [.hcommon.Empty](#hcommon-Empty) | [TunnelResponse](#tunnelservice-TunnelResponse) |  |
| Exit | [.hcommon.Empty](#hcommon-Empty) | [TunnelResponse](#tunnelservice-TunnelResponse) |  |

 



<a name="v2_hiddifyoptions_hiddify_options-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/hiddifyoptions/hiddify_options.proto



<a name="hiddifyoptions-DNSOptions"></a>

### DNSOptions
DNSOptions defines DNS-related configuration options.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| remote_dns_address | [string](#string) |  | Remote DNS server address. |
| remote_dns_domain_strategy | [DomainStrategy](#hiddifyoptions-DomainStrategy) |  | Strategy for resolving domains with remote DNS. |
| direct_dns_address | [string](#string) |  | Direct DNS server address. |
| direct_dns_domain_strategy | [DomainStrategy](#hiddifyoptions-DomainStrategy) |  | Strategy for resolving domains with direct DNS. |
| independent_dns_cache | [bool](#bool) |  | If true, enables independent DNS caching. |
| enable_fake_dns | [bool](#bool) |  | If true, enables fake DNS responses. |
| enable_dns_routing | [bool](#bool) |  | If true, enables DNS routing. |






<a name="hiddifyoptions-HiddifyOptions"></a>

### HiddifyOptions
HiddifyOptions defines the configuration options for the Hiddify application.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable_full_config | [bool](#bool) |  | Enables full configuration options. |
| log_level | [string](#string) |  | Specifies the logging level (e.g., INFO, DEBUG). |
| log_file | [string](#string) |  | Path to the log file. |
| enable_clash_api | [bool](#bool) |  | Indicates whether the Clash API is enabled. |
| clash_api_port | [uint32](#uint32) |  | Port for the Clash API (using uint32 for compatibility). |
| web_secret | [string](#string) |  | Secret key for accessing the Clash API. |
| region | [string](#string) |  | Region for the application. |
| block_ads | [bool](#bool) |  | If true, blocks ads. |
| use_xray_core_when_possible | [bool](#bool) |  | If true, use XRay core when possible. |
| rules | [Rule](#hiddifyoptions-Rule) | repeated | List of routing rules for traffic management. |
| warp | [WarpOptions](#hiddifyoptions-WarpOptions) |  | Configuration options for Warp. |
| warp2 | [WarpOptions](#hiddifyoptions-WarpOptions) |  | Additional configuration options for a second Warp instance. |
| mux | [MuxOptions](#hiddifyoptions-MuxOptions) |  | Configuration options for multiplexing. |
| tls_tricks | [TLSTricks](#hiddifyoptions-TLSTricks) |  | Options for TLS tricks. |
| dns_options | [DNSOptions](#hiddifyoptions-DNSOptions) |  | DNS-related options. |
| inbound_options | [InboundOptions](#hiddifyoptions-InboundOptions) |  | Inbound connection options. |
| url_test_options | [URLTestOptions](#hiddifyoptions-URLTestOptions) |  | URL test configuration options. |
| route_options | [RouteOptions](#hiddifyoptions-RouteOptions) |  | Routing-related options. |






<a name="hiddifyoptions-InboundOptions"></a>

### InboundOptions
InboundOptions defines the configuration options for inbound connections.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable_tun | [bool](#bool) |  | If true, enables TUN interface. |
| enable_tun_service | [bool](#bool) |  | If true, enables TUN service. |
| set_system_proxy | [bool](#bool) |  | If true, sets the system proxy. |
| mixed_port | [uint32](#uint32) |  | Port for mixed traffic (using uint32 for compatibility). |
| tproxy_port | [uint32](#uint32) |  | Port for TProxy connections (using uint32 for compatibility). |
| local_dns_port | [uint32](#uint32) |  | Port for local DNS service (using uint32 for compatibility). |
| mtu | [uint32](#uint32) |  | Maximum Transmission Unit size (using uint32 for compatibility). |
| strict_route | [bool](#bool) |  | If true, enforces strict routing. |
| tun_stack | [string](#string) |  | Specifies the TUN stack to use. |






<a name="hiddifyoptions-IntRange"></a>

### IntRange
IntRange defines a range of integers for various configurations.
It includes the starting and ending values of the range.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| from | [int32](#int32) |  | Starting value of the range. |
| to | [int32](#int32) |  | Ending value of the range. |






<a name="hiddifyoptions-MuxOptions"></a>

### MuxOptions
MuxOptions defines options for multiplexing connections.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable | [bool](#bool) |  | If true, enables multiplexing. |
| padding | [bool](#bool) |  | If true, enables padding for multiplexed connections. |
| max_streams | [int32](#int32) |  | Maximum number of streams allowed (using int32). |
| protocol | [string](#string) |  | Protocol used for multiplexing. |






<a name="hiddifyoptions-RouteOptions"></a>

### RouteOptions
RouteOptions defines options related to traffic routing.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resolve_destination | [bool](#bool) |  | If true, resolves the destination address. |
| ipv6_mode | [DomainStrategy](#hiddifyoptions-DomainStrategy) |  | Strategy for handling IPv6 addresses. |
| bypass_lan | [bool](#bool) |  | If true, bypasses LAN connections. |
| allow_connection_from_lan | [bool](#bool) |  | If true, allows connections from LAN. |






<a name="hiddifyoptions-Rule"></a>

### Rule
Rule defines routing rules for managing traffic.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_set_url | [string](#string) |  | URL of the rule set. |
| domains | [string](#string) |  | List of domains affected by this rule. |
| ip | [string](#string) |  | IP address associated with this rule. |
| port | [string](#string) |  | Port number associated with this rule. |
| network | [string](#string) |  | Network type (e.g., IPv4, IPv6). |
| protocol | [string](#string) |  | Protocol type (e.g., TCP, UDP). |
| outbound | [string](#string) |  | Outbound traffic handling (e.g., allow, deny). |






<a name="hiddifyoptions-TLSTricks"></a>

### TLSTricks
TLSTricks defines options for TLS tricks to obfuscate traffic.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable_fragment | [bool](#bool) |  | If true, enables fragmentation of packets. |
| fragment_size | [IntRange](#hiddifyoptions-IntRange) |  | Size of fragments to be used. |
| fragment_sleep | [IntRange](#hiddifyoptions-IntRange) |  | Sleep time between fragments. |
| mixed_sni_case | [bool](#bool) |  | If true, enables mixed SNI case for obfuscation. |
| enable_padding | [bool](#bool) |  | If true, enables padding of packets. |
| padding_size | [IntRange](#hiddifyoptions-IntRange) |  | Size of padding to be used. |






<a name="hiddifyoptions-URLTestOptions"></a>

### URLTestOptions
URLTestOptions defines the configuration options for URL testing.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| connection_test_url | [string](#string) |  | URL used for connection testing. |
| url_test_interval | [int64](#int64) |  | Interval for URL tests in milliseconds. |






<a name="hiddifyoptions-WarpAccount"></a>

### WarpAccount
WarpAccount defines account details for Warp.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account_id | [string](#string) |  | Unique account identifier. |
| access_token | [string](#string) |  | Access token for the account. |






<a name="hiddifyoptions-WarpOptions"></a>

### WarpOptions
WarpOptions defines configuration options for Warp.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique identifier for the Warp configuration. |
| enable_warp | [bool](#bool) |  | If true, enables Warp functionality. |
| mode | [string](#string) |  | Operating mode for Warp. |
| wireguard_config | [WarpWireguardConfig](#hiddifyoptions-WarpWireguardConfig) |  | Configuration for WireGuard (defined elsewhere). |
| fake_packets | [string](#string) |  | Fake packet configuration. |
| fake_packet_size | [IntRange](#hiddifyoptions-IntRange) |  | Size of fake packets. |
| fake_packet_delay | [IntRange](#hiddifyoptions-IntRange) |  | Delay for sending fake packets. |
| fake_packet_mode | [string](#string) |  | Mode for sending fake packets. |
| clean_ip | [string](#string) |  | Clean IP address to use. |
| clean_port | [uint32](#uint32) |  | Port for clean traffic (using uint32 for compatibility). |
| account | [WarpAccount](#hiddifyoptions-WarpAccount) |  | Account details for Warp (defined elsewhere). |






<a name="hiddifyoptions-WarpWireguardConfig"></a>

### WarpWireguardConfig
WarpWireguardConfig defines the configuration details for WireGuard.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| private_key | [string](#string) |  | Private key for WireGuard. |
| local_address_ipv4 | [string](#string) |  | Local IPv4 address for WireGuard. |
| local_address_ipv6 | [string](#string) |  | Local IPv6 address for WireGuard. |
| peer_public_key | [string](#string) |  | Peer public key for WireGuard. |
| client_id | [string](#string) |  | Client identifier for WireGuard. |





 


<a name="hiddifyoptions-DomainStrategy"></a>

### DomainStrategy
DomainStrategy defines the strategies for IP address preference when resolving domain names.

| Name | Number | Description |
| ---- | ------ | ----------- |
| as_is | 0 | As it is. |
| prefer_ipv4 | 1 | Prefer IPv4 addresses. |
| prefer_ipv6 | 2 | Prefer IPv6 addresses. |
| ipv4_only | 3 | Only use IPv4 addresses. |
| ipv6_only | 4 | Only use IPv6 addresses. |


 

 

 



<a name="v2_profile_profile_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/profile/profile_service.proto



<a name="profile-AddProfileRequest"></a>

### AddProfileRequest
AddProfileRequest is the request message for adding a profile
via URL or content.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The URL of the profile to add. |
| content | [string](#string) |  | The profile content to add (used if &#39;url&#39; is empty). |
| name | [string](#string) |  | The optional name of the profile. |
| mark_as_active | [bool](#bool) |  | Whether to mark the profile as active. |






<a name="profile-MultiProfilesResponse"></a>

### MultiProfilesResponse
MultiProfilesResponse is the response message for fetching multi profiles.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profiles | [ProfileEntity](#profile-ProfileEntity) | repeated | A list of profile entities. |
| response_code | [hcommon.ResponseCode](#hcommon-ResponseCode) |  | The response code indicating success or failure. |
| message | [string](#string) |  | A message indicating the result or error, if any. |






<a name="profile-ProfileRequest"></a>

### ProfileRequest
ProfileRequest is the request message for fetching or identifying
a profile by ID, name, or URL.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The ID of the profile to fetch (Fastest and recommended). |
| name | [string](#string) |  | The name of the profile to fetch (if both &#39;id&#39; and &#39;url&#39; are empty). |
| url | [string](#string) |  | The URL of the profile to fetch (if both &#39;id&#39; and &#39;name&#39; are empty). |






<a name="profile-ProfileResponse"></a>

### ProfileResponse
ProfileResponse is the response message for profile service operations.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [ProfileEntity](#profile-ProfileEntity) |  | The profile entity, populated in successful operations. |
| response_code | [hcommon.ResponseCode](#hcommon-ResponseCode) |  | The response code indicating success or failure. |
| message | [string](#string) |  | A message indicating the result or error, if any. |





 

 

 


<a name="profile-ProfileService"></a>

### ProfileService
ProfileService defines the RPC methods available for managing profiles.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetProfile | [ProfileRequest](#profile-ProfileRequest) | [ProfileResponse](#profile-ProfileResponse) | GetProfile fetches a profile by ID, name, or URL. |
| UpdateProfile | [ProfileEntity](#profile-ProfileEntity) | [ProfileResponse](#profile-ProfileResponse) | UpdateProfile updates an existing profile. |
| GetAllProfiles | [.hcommon.Empty](#hcommon-Empty) | [MultiProfilesResponse](#profile-MultiProfilesResponse) | GetAllProfiles fetches all profiles. |
| GetActiveProfile | [.hcommon.Empty](#hcommon-Empty) | [ProfileResponse](#profile-ProfileResponse) | GetActiveProfile retrieves the currently active profile. |
| SetActiveProfile | [ProfileRequest](#profile-ProfileRequest) | [.hcommon.Response](#hcommon-Response) | SetActiveProfile sets a profile as active, identified by ID, name, or URL. |
| AddProfile | [AddProfileRequest](#profile-AddProfileRequest) | [ProfileResponse](#profile-ProfileResponse) | AddProfile adds a new profile using either a URL or the raw profile content. |
| DeleteProfile | [ProfileRequest](#profile-ProfileRequest) | [.hcommon.Response](#hcommon-Response) | DeleteProfile deletes a profile identified by ID, name, or URL. |

 



<a name="v2_profile_profile-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/profile/profile.proto



<a name="profile-ProfileEntity"></a>

### ProfileEntity
ProfileEntity defines a profile entity.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique identifier for the profile. |
| name | [string](#string) |  | bool active = 2; // Indicates if the profile is active.

Name of the profile. |
| url | [string](#string) |  | URL associated with the profile. |
| last_update | [int64](#int64) |  | Last update time in milliseconds of the profile. |
| options | [ProfileOptions](#profile-ProfileOptions) |  | Options associated with the profile. |
| sub_info | [SubscriptionInfo](#profile-SubscriptionInfo) |  | Subscription-related information. |
| override_hiddify_options | [hiddifyoptions.HiddifyOptions](#hiddifyoptions-HiddifyOptions) |  | Override Hiddify options. |






<a name="profile-ProfileOptions"></a>

### ProfileOptions
ProfileOptions defines options for a profile.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| update_interval | [int64](#int64) |  | Update interval in milliseconds. |






<a name="profile-SubscriptionInfo"></a>

### SubscriptionInfo
SubscriptionInfo defines subscription-related information.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| upload | [int64](#int64) |  | Upload speed in bytes. |
| download | [int64](#int64) |  | Download speed in bytes. |
| total | [int64](#int64) |  | Total data in bytes. |
| expire | [int64](#int64) |  | Expiration time in milliseconds of the subscription. |
| web_page_url | [string](#string) |  | URL for the web page. |
| support_url | [string](#string) |  | URL for support. |





 

 

 

 



<a name="v2_hcommon_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/hcommon/common.proto



<a name="hcommon-Empty"></a>

### Empty







<a name="hcommon-Response"></a>

### Response



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [ResponseCode](#hcommon-ResponseCode) |  |  |
| message | [string](#string) |  |  |





 


<a name="hcommon-ResponseCode"></a>

### ResponseCode


| Name | Number | Description |
| ---- | ------ | ----------- |
| OK | 0 |  |
| FAILED | 1 |  |
| AUTH_NEED | 2 |  |


 

 

 



<a name="v2_hello_hello_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/hello/hello_service.proto


 

 

 


<a name="hello-Hello"></a>

### Hello


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SayHello | [HelloRequest](#hello-HelloRequest) | [HelloResponse](#hello-HelloResponse) |  |
| SayHelloStream | [HelloRequest](#hello-HelloRequest) stream | [HelloResponse](#hello-HelloResponse) stream |  |

 



<a name="v2_hello_hello-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v2/hello/hello.proto



<a name="hello-HelloRequest"></a>

### HelloRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






<a name="hello-HelloResponse"></a>

### HelloResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| message | [string](#string) |  |  |





 

 

 

 



<a name="extension_extension_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## extension/extension_service.proto


 

 

 


<a name="extension-ExtensionHostService"></a>

### ExtensionHostService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListExtensions | [.hcommon.Empty](#hcommon-Empty) | [ExtensionList](#extension-ExtensionList) |  |
| Connect | [ExtensionRequest](#extension-ExtensionRequest) | [ExtensionResponse](#extension-ExtensionResponse) stream |  |
| EditExtension | [EditExtensionRequest](#extension-EditExtensionRequest) | [ExtensionActionResult](#extension-ExtensionActionResult) |  |
| SubmitForm | [SendExtensionDataRequest](#extension-SendExtensionDataRequest) | [ExtensionActionResult](#extension-ExtensionActionResult) |  |
| Close | [ExtensionRequest](#extension-ExtensionRequest) | [ExtensionActionResult](#extension-ExtensionActionResult) |  |
| GetUI | [ExtensionRequest](#extension-ExtensionRequest) | [ExtensionActionResult](#extension-ExtensionActionResult) |  |

 



<a name="extension_extension-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## extension/extension.proto



<a name="extension-EditExtensionRequest"></a>

### EditExtensionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| extension_id | [string](#string) |  |  |
| enable | [bool](#bool) |  |  |






<a name="extension-ExtensionActionResult"></a>

### ExtensionActionResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| extension_id | [string](#string) |  |  |
| code | [hcommon.ResponseCode](#hcommon-ResponseCode) |  |  |
| message | [string](#string) |  |  |






<a name="extension-ExtensionList"></a>

### ExtensionList



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| extensions | [ExtensionMsg](#extension-ExtensionMsg) | repeated |  |






<a name="extension-ExtensionMsg"></a>

### ExtensionMsg



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| enable | [bool](#bool) |  |  |






<a name="extension-ExtensionRequest"></a>

### ExtensionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| extension_id | [string](#string) |  |  |
| data | [ExtensionRequest.DataEntry](#extension-ExtensionRequest-DataEntry) | repeated |  |






<a name="extension-ExtensionRequest-DataEntry"></a>

### ExtensionRequest.DataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="extension-ExtensionResponse"></a>

### ExtensionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ExtensionResponseType](#extension-ExtensionResponseType) |  |  |
| extension_id | [string](#string) |  |  |
| json_ui | [string](#string) |  |  |






<a name="extension-SendExtensionDataRequest"></a>

### SendExtensionDataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| extension_id | [string](#string) |  |  |
| button | [string](#string) |  |  |
| data | [SendExtensionDataRequest.DataEntry](#extension-SendExtensionDataRequest-DataEntry) | repeated |  |






<a name="extension-SendExtensionDataRequest-DataEntry"></a>

### SendExtensionDataRequest.DataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 


<a name="extension-ExtensionResponseType"></a>

### ExtensionResponseType


| Name | Number | Description |
| ---- | ------ | ----------- |
| NOTHING | 0 |  |
| UPDATE_UI | 1 |  |
| SHOW_DIALOG | 2 |  |
| END | 3 |  |


 

 

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

