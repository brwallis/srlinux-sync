package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"
    "strconv"

	"encoding/json"

	// "github.com/google/gnxi/utils"
	"github.com/google/gnxi/utils/xpath"
	// "github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	srl "srl-apps/pkg/api/v19.11.5/nokia.com/srlinux/sdk/protos"
	yang "srl-apps/pkg/api/v19.11.5/yang"
)

// CfgTranxEntry NDK transaction
type CfgTranxEntry struct {
	Op   srl.SdkMgrOperation
	Key  *srl.ConfigKey
	Data *string
}

// YangFumble YANG structure
type YangFumble struct {
	TrafficDeltaThreshold struct {
		Value string `json:"value"`
	} `json:"traffic_delta_threshold"`
	TrafficDeltaLast struct {
		Value uint8 `json:"value"`
	} `json:"traffic_delta_last"`
	AlarmState struct {
		Value string `json:"value"`
	} `json:"alarm_state"`
	TrafficTotalInBps struct {
		Value uint64 `json:"value"`
	} `json:"traffic_total_in_bps"`
	TrafficTotalOutBps struct {
		Value uint64 `json:"value"`
	} `json:"traffic_total_out_bps"`
}

// GoFumbleMgr base instance
type GoFumbleMgr struct {
	sync.RWMutex

	OwnAppID uint32
	StreamID uint64
	Client   srl.SdkMgrServiceClient
	GrpcConn *grpc.ClientConn
	Wg       sync.WaitGroup

	CfgTranxMap map[string][]CfgTranxEntry

	YangFumble YangFumble
}

// Duration just wraps time.Duration
type Duration struct {
	Duration time.Duration
}

// TelemetryGNMI plugin instance
type TelemetryGNMI struct {
	Addresses     []string
	Subscriptions []Subscription

	// Optional subscription configuration
	Encoding    string
	Origin      string
	Prefix      string
	Target      string
	UpdatesOnly bool

	Username string
	Password string

	// Redial
	Redial Duration

	// GRPC TLS settings
	EnableTLS bool
	// internaltls.ClientConfig

	// Internal state
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Subscription for a GNMI client
type Subscription struct {
	Name   string
	Origin string
	Path   string

	// Subscription mode and interval
	SubscriptionMode string
	SampleInterval   Duration

	// Duplicate suppression
	SuppressRedundant bool
	HeartbeatInterval Duration
}

const (
	address   = "localhost:50053"
	agentName = "fumble"
)

var (
	// FumbleMgr instance
	FumbleMgr GoFumbleMgr
)

// UpdateTelemetry updates the NDK with a JS path and data
func UpdateTelemetry(JsPath *string, JsData *string) {
	ctx := context.Background()

	// Set up agent name
	ctx = metadata.AppendToOutgoingContext(ctx, "agent_name", agentName)

	telClient := srl.NewSdkMgrTelemetryServiceClient(FumbleMgr.GrpcConn)

	key := &srl.TelemetryKey{JsPath: *JsPath}
	data := &srl.TelemetryData{JsonContent: *JsData}
	entry := &srl.TelemetryInfo{Key: key, Data: data}
	telReq := &srl.TelemetryUpdateRequest{}
	telReq.State = make([]*srl.TelemetryInfo, 0)
	telReq.State = append(telReq.State, entry)

	r1, err := telClient.TelemetryAddOrUpdate(ctx, telReq)
	if err != nil {
		log.Fatalf("Could not update telemetry for key : %s", *JsPath)
	}
	log.Printf("Telemetry add/update status: %s error_string: %s", r1.GetStatus(), r1.GetErrorStr())
}

// DeleteTelemetry updates the NDK with a JS path and data
func DeleteTelemetry(JsPath *string) {
	ctx := context.Background()

	// Set up agent name
	ctx = metadata.AppendToOutgoingContext(ctx, "agent_name", agentName)

	telClient := srl.NewSdkMgrTelemetryServiceClient(FumbleMgr.GrpcConn)

	key := &srl.TelemetryKey{JsPath: *JsPath}
	telReq := &srl.TelemetryDeleteRequest{}
	telReq.Key = make([]*srl.TelemetryKey, 0)
	telReq.Key = append(telReq.Key, key)

	r1, err := telClient.TelemetryDelete(ctx, telReq)
	if err != nil {
		log.Fatalf("Could not delete telemetry for key : %s", *JsPath)
	}
	log.Printf("Telemetry delete status: %s error_string: %s", r1.GetStatus(), r1.GetErrorStr())
}

// HandleGoFumbleConfigEvent takes a config event from the NDK and processes it
func HandleGoFumbleConfigEvent(op srl.SdkMgrOperation, key *srl.ConfigKey, data *string) {
	log.Printf("\n jspath %s keys %v", key.GetJsPath(), key.GetKeys())
	JsPath := fmt.Sprintf(".system.fumble")

	if data != nil {
		log.Printf("\n data %v", *data)
	}

	if data == nil {
		log.Printf("\nNo data found")
		if op == srl.SdkMgrOperation_Delete {
			log.Printf("\nDelete operation")
			DeleteTelemetry(&JsPath)
		}
		return
	}

	var cur YangFumble
	if err := json.Unmarshal([]byte(*data), &cur); err != nil {
		log.Fatalf("Can not unmarshal config data: %s error %s", *data, err)
	}

	FumbleMgr.YangFumble = cur
	log.Printf("\nkey %v doing something now", *key)

	YangData := YangFumble{}
	YangData.TrafficDeltaThreshold.Value = FumbleMgr.YangFumble.TrafficDeltaThreshold.Value
	//if len(FumbleMgr.YangFumble.TrafficDeltaThreshold.Value) < 1 {
	//	YangData.TrafficDeltaLast.Value = fmt.Sprintf("0")
	//} else {
	//	YangData.TrafficDeltaLast.Value = FumbleMgr.YangFumble.TrafficDeltaThreshold.Value
	//}
	//YangData.AlarmState.Value = "active"
	JsData, err := json.Marshal(YangData)
	if err != nil {
		log.Fatalf("Can not marshal config data:%v error %s", *data, err)
	}
	JsString := string(JsData)
	log.Printf("JsPath: %s", JsPath)
	log.Printf("JsString: %s", JsString)

	UpdateTelemetry(&JsPath, &JsString)

}

// HandleConfigEvent handles a configuration event
func HandleConfigEvent(op srl.SdkMgrOperation, key *srl.ConfigKey, data *string) {
	log.Printf("\nkey %v", *key)

	if key.GetJsPath() != ".commit.end" {
		FumbleMgr.CfgTranxMap[key.GetJsPath()] = append(FumbleMgr.CfgTranxMap[key.GetJsPath()], CfgTranxEntry{Op: op, Key: key, Data: data})
		return
	}

	// for _, item := range FibMgr.CfgTranxMap[".gofib.ipv4_routes"] {
	// 	HandleIpv4ConfigEvent(item.Op, item.Key, item.Data)
	// }

	for _, item := range FumbleMgr.CfgTranxMap[".system.fumble"] {
		HandleGoFumbleConfigEvent(item.Op, item.Key, item.Data)
	}

	// Delete all current candidate list.
	FumbleMgr.CfgTranxMap = make(map[string][]CfgTranxEntry)
}

// HandleNotificationEvent handles a notification event
func HandleNotificationEvent(in *srl.NotificationStreamResponse) {
	for _, item := range in.Notification {
		switch x := item.SubscriptionTypes.(type) {
		case *srl.Notification_Config:
			resp := item.GetConfig()
			if resp.Data != nil {
				HandleConfigEvent(resp.Op, resp.Key, &resp.Data.Json)
			} else {
				HandleConfigEvent(resp.Op, resp.Key, nil)
			}
		default:
			log.Printf("\nGot unhandled message %s ", x)
		}
	}
}

// SubscribeStreams subscribes to all notifications
func SubscribeStreams() {
	ctx := context.Background()
	// Set up agent name
	ctx = metadata.AppendToOutgoingContext(ctx, "agent_name", agentName)

	notifRegReq := &srl.NotificationRegisterRequest{Op: srl.NotificationRegisterRequest_Create}
	r3, err := FumbleMgr.Client.NotificationRegister(ctx, notifRegReq)
	if err != nil {
		log.Fatalf("Could not register for notification : %v", err)
	}
	log.Printf("Notification registration status : %s stream_id %v\n", r3.Status, r3.GetStreamId())

	FumbleMgr.StreamID = r3.GetStreamId()

	cfgEntry := &srl.NotificationRegisterRequest_Config{Config: &srl.ConfigSubscriptionRequest{}}
	cfgReq := &srl.NotificationRegisterRequest{Op: srl.NotificationRegisterRequest_AddSubscription, StreamId: r3.GetStreamId(), SubscriptionTypes: cfgEntry}
	r4, err := FumbleMgr.Client.NotificationRegister(ctx, cfgReq)
	if err != nil {
		log.Fatalf("Could not register for config notification : %v", err)
	}
	log.Printf("Config notification registration status : %s stream_id %v\n", r4.Status, r4.GetStreamId())
}

// RunRecvNotification is called when a notification is received
func RunRecvNotification(wg *sync.WaitGroup) {
	defer wg.Done()

	ctx := context.Background()

	// Set up agent name
	ctx = metadata.AppendToOutgoingContext(ctx, "agent_name", agentName)

	notif_client := srl.NewSdkNotificationServiceClient(FumbleMgr.GrpcConn)

	subReq := &srl.NotificationStreamRequest{StreamId: FumbleMgr.StreamID}

	stream, err := notif_client.NotificationStream(ctx, subReq)

	if err != nil {
		log.Fatalf("Could not subscribe for notification : %v", err)
	}

	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive a notification : %v", err)
			}
			HandleNotificationEvent(in)
		}
	}()
	<-waitc
}

// FumbleInit initializes the agent
func FumbleInit() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	// Set up base service client
	client := srl.NewSdkMgrServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set up agent name
	ctx = metadata.AppendToOutgoingContext(ctx, "agent_name", agentName)

	// Register agent with NDK manager.
	r, err := client.AgentRegister(ctx, &srl.AgentRegistrationRequest{})
	if err != nil {
		log.Fatalf("Could not register: %v", err)
	}
	log.Printf("Agent registration status: %s AppId: %d\n", r.Status, r.GetAppId())

	FumbleMgr = GoFumbleMgr{
		GrpcConn: conn,
		Client:   client,
		OwnAppID: r.GetAppId(),
	}

	FumbleMgr.CfgTranxMap = make(map[string][]CfgTranxEntry)
}

type InterfaceStats struct {
	Interface []struct {
		Name        string `json:"name"`
		TrafficRate struct {
			InBps  string `json:"in-bps"`
			OutBps string `json:"out-bps"`
		} `json:"traffic-rate"`
	} `json:"interface"`
}

func PollInterfaceStats() {
	for {
		dev := GnmiSrlDeviceInterfaceGet()
		var sum_inbps uint64 = 0
		var sum_outbps uint64 = 0
		for _, v := range dev.Interface {
			if v.TrafficRate != nil {
				in_bps := v.TrafficRate.InBps
				out_bps := v.TrafficRate.OutBps
				if in_bps != nil {
					sum_inbps = sum_inbps + *in_bps
				}
				if out_bps != nil {
					sum_outbps = sum_outbps + *out_bps
				}
			}
		}
		fmt.Print("Total", sum_inbps, sum_outbps)

		time.Sleep(5 * time.Second)
	}
}

func GnmiSrlDeviceInterfaceGet() *yang.Device {
	targetAddr := "unix:///opt/srlinux/var/run/sr_gnmi_server"
	conn, err := grpc.Dial(targetAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Dialing to %q failed: %v", targetAddr, err)
	}
	defer conn.Close()

	cli := gpb.NewGNMIClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	encoding, ok := gpb.Encoding_value["JSON_IETF"]
	if !ok {
		var gnmiEncodingList []string
		for _, name := range gpb.Encoding_name {
			gnmiEncodingList = append(gnmiEncodingList, name)
		}
		log.Fatalf("Supported encodings: %s", strings.Join(gnmiEncodingList, ", "))
	}

	var pbPathList []*gpb.Path
	var pbModelDataList []*gpb.ModelData

	xPathFlags := [1]string{"/interface[name=*]"}
	for _, xPath := range xPathFlags {
		pbPath, err := xpath.ToGNMIPath(xPath)
		if err != nil {
			log.Fatalf("error in parsing xpath %q to gnmi path", xPath)
		}
		pbPathList = append(pbPathList, pbPath)
	}

	getRequest := &gpb.GetRequest{
		Encoding:  gpb.Encoding(encoding),
		Path:      pbPathList,
		UseModels: pbModelDataList,
	}

	getResponse, err := cli.Get(ctx, getRequest)
	if err != nil {
		log.Fatalf("Get failed: %v", err)
	}

	dev := &yang.Device{}

	err = yang.Unmarshal(getResponse.GetNotification()[0].GetUpdate()[0].GetVal().GetJsonIetfVal(), dev)
	if err != nil {
		log.Fatalf("Error %v", err)
	}

	return dev
}

func GnmiSrlDeviceInterfaceSubscribe() {
	targetAddr := "unix:///opt/srlinux/var/run/sr_gnmi_server"
	conn, err := grpc.Dial(targetAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Dialing to %q failed: %v", targetAddr, err)
	}
	defer conn.Close()

	cli := gpb.NewGNMIClient(conn)

	ctx := context.Background()

	encoding, ok := gpb.Encoding_value["JSON_IETF"]
	if !ok {
		var gnmiEncodingList []string
		for _, name := range gpb.Encoding_name {
			gnmiEncodingList = append(gnmiEncodingList, name)
		}
		log.Fatalf("Supported encodings: %s", strings.Join(gnmiEncodingList, ", "))
	}

	path, err := xpath.ToGNMIPath("/interface[name=*]")
	if err != nil {
		log.Fatalf("error in parsing xpath %q to gnmi path", "/interface[name=*]")
	}

	subsList := []*gpb.Subscription{
		&gpb.Subscription{
			Path: path,
		},
	}

	req := &gpb.SubscribeRequest_Subscribe{
		Subscribe: &gpb.SubscriptionList{
			Encoding:     gpb.Encoding(encoding),
			Subscription: subsList,
		},
	}

	gnmiSubscribeRequest := &gpb.SubscribeRequest{
		Request: req,
	}

	stream, err := cli.Subscribe(ctx)
	if err != nil {
		log.Fatalf("Get failed: %v", err)
	}

	if err := stream.Send(gnmiSubscribeRequest); err != nil {
		log.Fatalf("Failed to send a subsribe request for interface: %v", err)
	}

	stats_map := make(map[string]*yang.SrlNokiaInterfaces_Interface_TrafficRate)
	var sum_inbps uint64 = 0
	var sum_outbps uint64 = 0

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("Failed to receive a stream message : %v", err)
		}
		// fmt.Printf("\n\n%v", in.GetResponse())

		switch x := in.GetResponse().(type) {
		case *gpb.SubscribeResponse_Update:
			resp := in.GetUpdate()

			for _, v := range resp.Update {
				for _, e := range v.GetPath().GetElem() {
					if e.GetName() == "traffic-rate" {
						fmt.Printf("\n Update--->")
						intf := &yang.SrlNokiaInterfaces_Interface_TrafficRate{}

						err = yang.Unmarshal(v.GetVal().GetJsonIetfVal(), intf)
						if err != nil {
							log.Fatalf("Error %v", err)
						}

						old, ok := stats_map[v.GetPath().String()]
						if ok {
							if old.InBps != nil {
								sum_inbps = sum_inbps - *old.InBps
							}
							if old.OutBps != nil {
								sum_outbps = sum_outbps - *old.OutBps
							}
						}

						stats_map[v.GetPath().String()] = intf

						fmt.Printf("\n path key: %s", v.GetPath().String())
						if intf.InBps != nil {
							sum_inbps = sum_inbps + *intf.InBps
							fmt.Printf("\n Total InBps: %v", sum_inbps)
						}
						if intf.OutBps != nil {
							sum_outbps = sum_outbps + *intf.OutBps
							fmt.Printf("\n Total OutBps: %v", sum_outbps)
						}
		//FumbleMgr.YangFumble.TrafficDeltaLast.Value = "5"
		FumbleMgr.YangFumble.TrafficTotalInBps.Value = sum_inbps
		FumbleMgr.YangFumble.TrafficTotalOutBps.Value = sum_outbps

		// Calculate deltas
		delta := int64(sum_inbps) - int64(sum_outbps)
		if delta >= 1 {
			var deltaPercentage = (uint64(delta) * 100 / sum_inbps)
			fmt.Printf("Delta is: %d\n", delta)
			fmt.Printf("Delta percent is: %d\n", deltaPercentage)
			FumbleMgr.YangFumble.TrafficDeltaLast.Value = uint8(deltaPercentage)
		} else {
			FumbleMgr.YangFumble.TrafficDeltaLast.Value = 0
		}

		// Convert string to int
		TrafficDeltaThresholdInt, err := strconv.ParseUint(FumbleMgr.YangFumble.TrafficDeltaThreshold.Value, 10, 8)
		if err != nil {
			panic(err)
		}
		// Check for breach
		if uint8(TrafficDeltaThresholdInt) < FumbleMgr.YangFumble.TrafficDeltaLast.Value {
			FumbleMgr.YangFumble.AlarmState.Value = "active"
		} else {
			FumbleMgr.YangFumble.AlarmState.Value = "inactive"
		}

		JsData, err := json.Marshal(FumbleMgr.YangFumble)
		if err != nil {
			log.Fatalf("Can not marshal config data")
		}
		JsPath := fmt.Sprintf(".system.fumble")
		JsString := string(JsData)
		log.Printf("JsPath: %s", JsPath)
		log.Printf("JsString: %s", JsString)

		FumbleMgr.YangFumble.AlarmState.Value = "active"
		// Update telemetry for totals
		UpdateTelemetry(&JsPath, &JsString)

		log.Printf("\nTotal: %s in, %s out", sum_inbps, sum_outbps)

					}
				}
			}
			for _, v := range resp.Delete {
				fmt.Printf("\n\n Delete---> %v", v)
			}
		default:
			log.Printf("\nGot unhandled message %s ", x)
		}
	}
}

func main() {

	FumbleInit()

	SubscribeStreams()

	FumbleMgr.Wg.Add(1)
	go RunRecvNotification(&FumbleMgr.Wg)

	// FumbleMgr.Wg.Add(1)
	// go PollInterfaceStats()

	FumbleMgr.Wg.Add(1)
	go GnmiSrlDeviceInterfaceSubscribe()

	FumbleMgr.Wg.Wait()

	FumbleMgr.GrpcConn.Close()
}
