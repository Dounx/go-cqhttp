package server

import (
	"context"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/coolq"
	pb "github.com/Mrs4s/go-cqhttp/proto"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
	"net"
)

type EventPusher struct {
	Pool map[chan *pb.Event]bool
}

func NewEventPusher() *EventPusher {
	return &EventPusher{
		Pool: make(map[chan *pb.Event]bool),
	}
}

func (ep *EventPusher) Registry(ch chan *pb.Event) {
	ep.Pool[ch] = true
}

func (ep *EventPusher) UnRegistry(ch chan *pb.Event) {
	delete(ep.Pool, ch)
}

func (ep *EventPusher) Publish(event *pb.Event) {
	for ch := range ep.Pool {
		ch <- event
	}
}

type rpcServer struct {
	Service     *APIService
	EventPusher *EventPusher
}

//RPCServer 初始化一个rpcServer实例
var RPCServer = &rpcServer{}

func (s *rpcServer) Run(addr, token string, bot *coolq.CQBot) {
	server := grpc.NewServer()

	eventPusher := NewEventPusher()
	s.EventPusher = eventPusher
	pb.RegisterAPIServer(server, &APIService{bot: bot, EventPusher: eventPusher})

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}

	log.Infof("CQ RPC 服务器已启动: %v", addr)

	bot.OnEventPush(s.onBotPushEvent)

	err = server.Serve(lis)
	if err != nil {
		log.Fatalf("server.Serve err: %v", err)
	}
}

func (s *rpcServer) onBotPushEvent(m coolq.MSG) {
	event := &pb.Event{}
	json, _ := json.MarshalToString(m)
	wrap := coolq.MSG{"payload": json}
	if err := fillStruct(wrap, event); err != nil {
		log.Info(err)
		return
	}
	s.EventPusher.Publish(event)
}

type CQError struct {
	RetCode int    `json:"retcode"`
	Msg     string `json:"msg"`
	Wording string `json:"wording"`
	Status  string `json:"status"`
}

func (e *CQError) Error() string {
	return e.Msg
}

type APIService struct {
	pb.UnimplementedAPIServer

	bot         *coolq.CQBot
	EventPusher *EventPusher
}

func (s *APIService) GetLoginInfo(ctx context.Context, request *pb.GetLoginInfoRequest) (*pb.GetLoginInfoResponse, error) {
	result := s.bot.CQGetLoginInfo()["data"].(coolq.MSG)
	response := &pb.GetLoginInfoResponse{}
	if err := fillStruct(result, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetFriendList(ctx context.Context, request *pb.GetFriendListRequest) (*pb.GetFriendListResponse, error) {
	result := s.bot.CQGetFriendList()["data"].([]coolq.MSG)

	wrap := coolq.MSG{"friends": result}
	response := &pb.GetFriendListResponse{}
	if err := fillStruct(wrap, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupList(ctx context.Context, request *pb.GetGroupListRequest) (*pb.GetGroupListResponse, error) {
	result := s.bot.CQGetGroupList(request.NoCache)["data"].([]coolq.MSG)

	wrap := coolq.MSG{"groups": result}
	response := &pb.GetGroupListResponse{}
	if err := fillStruct(wrap, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupInfo(ctx context.Context, request *pb.GetGroupInfoRequest) (*pb.GetGroupInfoResponse, error) {
	result := s.bot.CQGetGroupInfo(request.GroupId, request.NoCache)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetGroupInfoResponse{}
	wrap := coolq.MSG{"group": data}
	if err := fillStruct(wrap, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupMemberList(ctx context.Context, request *pb.GetGroupMemberListRequest) (*pb.GetGroupMemberListResponse, error) {
	result := s.bot.CQGetGroupMemberList(request.GroupId, request.NoCache)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].([]coolq.MSG)
	response := &pb.GetGroupMemberListResponse{}
	wrap := coolq.MSG{"group_members": data}
	if err := fillStruct(wrap, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupMemberInfo(ctx context.Context, request *pb.GetGroupMemberInfoRequest) (*pb.GetGroupMemberInfoResponse, error) {
	result := s.bot.CQGetGroupMemberInfo(request.GroupId, request.UserId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetGroupMemberInfoResponse{}
	wrap := coolq.MSG{"group_member": data}
	if err := fillStruct(wrap, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupFileSystemInfo(ctx context.Context, request *pb.GetGroupFileSystemInfoRequest) (*pb.GetGroupFileSystemInfoResponse, error) {
	result := s.bot.CQGetGroupFileSystemInfo(request.GroupId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(*client.GroupFileSystem)
	response := &pb.GetGroupFileSystemInfoResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupRootFiles(ctx context.Context, request *pb.GetGroupRootFilesRequest) (*pb.GetGroupRootFilesResponse, error) {
	result := s.bot.CQGetGroupRootFiles(request.GroupId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetGroupRootFilesResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupFilesByFolder(ctx context.Context, request *pb.GetGroupFilesByFolderRequest) (*pb.GetGroupFilesByFolderResponse, error) {
	result := s.bot.CQGetGroupFilesByFolderId(request.GroupId, request.FolderId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetGroupFilesByFolderResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupFileUrl(ctx context.Context, request *pb.GetGroupFileUrlRequest) (*pb.GetGroupFileUrlResponse, error) {
	result := s.bot.CQGetGroupFileUrl(request.GroupId, request.FileId, request.BusId)["data"].(coolq.MSG)
	response := &pb.GetGroupFileUrlResponse{}
	if err := fillStruct(result, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetWordSlices(ctx context.Context, request *pb.GetWordSlicesRequest) (*pb.GetWordSlicesResponse, error) {
	result := s.bot.CQGetWordSlices(request.Content)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetWordSlicesResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) SendGroupMessage(ctx context.Context, request *pb.SendGroupMessageRequest) (*pb.SendGroupMessageResponse, error) {
	message := gjson.Parse(request.Message)
	result := s.bot.CQSendGroupMessage(request.GroupId, message, request.AutoEscape)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.SendGroupMessageResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) SendGroupForwardMessage(ctx context.Context, request *pb.SendGroupForwardMessageRequest) (*pb.SendGroupForwardMessageResponse, error) {
	messages := gjson.Parse(request.Messages)
	result := s.bot.CQSendGroupForwardMessage(request.GroupId, messages)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.SendGroupForwardMessageResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) SendPrivateMessage(ctx context.Context, request *pb.SendPrivateMessageRequest) (*pb.SendPrivateMessageResponse, error) {
	message := gjson.Parse(request.Message)
	result := s.bot.CQSendPrivateMessage(request.UserId, message, request.AutoEscape)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.SendPrivateMessageResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) SetGroupCard(ctx context.Context, request *pb.SetGroupCardRequest) (*pb.SetGroupCardResponse, error) {
	result := s.bot.CQSetGroupCard(request.GroupId, request.UserId, request.Card)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupCardResponse{}, nil
}

func (s *APIService) SetGroupSpecialTitle(ctx context.Context, request *pb.SetGroupSpecialTitleRequest) (*pb.SetGroupSpecialTitleResponse, error) {
	result := s.bot.CQSetGroupSpecialTitle(request.GroupId, request.UserId, request.SpecialTitle)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupSpecialTitleResponse{}, nil
}

func (s *APIService) SetGroupName(ctx context.Context, request *pb.SetGroupNameRequest) (*pb.SetGroupNameResponse, error) {
	result := s.bot.CQSetGroupName(request.GroupId, request.GroupName)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupNameResponse{}, nil
}

func (s *APIService) SetGroupNotice(ctx context.Context, request *pb.SetGroupNoticeRequest) (*pb.SetGroupNoticeResponse, error) {
	result := s.bot.CQSetGroupMemo(request.GroupId, request.Content)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupNoticeResponse{}, nil
}

func (s *APIService) SetGroupKick(ctx context.Context, request *pb.SetGroupKickRequest) (*pb.SetGroupKickResponse, error) {
	result := s.bot.CQSetGroupKick(request.GroupId, request.UserId, request.Message, request.RejectAddRequest)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupKickResponse{}, nil
}

func (s *APIService) SetGroupBan(ctx context.Context, request *pb.SetGroupBanRequest) (*pb.SetGroupBanResponse, error) {
	result := s.bot.CQSetGroupBan(request.GroupId, request.UserId, request.Duration)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupBanResponse{}, nil
}

func (s *APIService) SetGroupWholeBan(ctx context.Context, request *pb.SetGroupWholeBanRequest) (*pb.SetGroupWholeBanResponse, error) {
	result := s.bot.CQSetGroupWholeBan(request.GroupId, request.Enable)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupWholeBanResponse{}, nil
}

func (s *APIService) SetGroupLeave(ctx context.Context, request *pb.SetGroupLeaveRequest) (*pb.SetGroupLeaveResponse, error) {
	result := s.bot.CQSetGroupLeave(request.GroupId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupLeaveResponse{}, nil
}

func (s *APIService) GetGroupAtAllRemain(ctx context.Context, request *pb.GetGroupAtAllRemainRequest) (*pb.GetGroupAtAllRemainResponse, error) {
	result := s.bot.CQGetAtAllRemain(request.GroupId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(*client.AtAllRemainInfo)
	response := &pb.GetGroupAtAllRemainResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) ProcessFriendRequest(ctx context.Context, request *pb.ProcessFriendRequestRequest) (*pb.ProcessFriendRequestResponse, error) {
	result := s.bot.CQProcessFriendRequest(request.Flag, request.Approve)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.ProcessFriendRequestResponse{}, nil
}

func (s *APIService) ProcessGroupRequest(ctx context.Context, request *pb.ProcessGroupRequestRequest) (*pb.ProcessGroupRequestResponse, error) {
	result := s.bot.CQProcessGroupRequest(request.Flag, request.SubType, request.Reason, request.Approve)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.ProcessGroupRequestResponse{}, nil
}

func (s *APIService) DeleteMessage(ctx context.Context, request *pb.DeleteMessageRequest) (*pb.DeleteMessageResponse, error) {
	result := s.bot.CQDeleteMessage(request.MessageId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.DeleteMessageResponse{}, nil
}

func (s *APIService) SetGroupAdmin(ctx context.Context, request *pb.SetGroupAdminRequest) (*pb.SetGroupAdminResponse, error) {
	result := s.bot.CQSetGroupAdmin(request.GroupId, request.UserId, request.Enable)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupAdminResponse{}, nil
}

func (s *APIService) GetVipInfo(ctx context.Context, request *pb.GetVipInfoRequest) (*pb.GetVipInfoResponse, error) {
	result := s.bot.CQGetVipInfo(request.UserId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetVipInfoResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupHonorInfo(ctx context.Context, request *pb.GetGroupHonorInfoRequest) (*pb.GetGroupHonorInfoResponse, error) {
	result := s.bot.CQGetGroupHonorInfo(request.GroupId, request.Type)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetGroupHonorInfoResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetStrangerInfo(ctx context.Context, request *pb.GetStrangerInfoRequest) (*pb.GetStrangerInfoResponse, error) {
	result := s.bot.CQGetStrangerInfo(request.UserId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetStrangerInfoResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) HandleQuickOperation(ctx context.Context, request *pb.HandleQuickOperationRequest) (*pb.HandleQuickOperationResponse, error) {
	result := s.bot.CQHandleQuickOperation(gjson.Parse(request.Context), gjson.Parse(request.Operation))
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.HandleQuickOperationResponse{}, nil
}

func (s *APIService) GetImage(ctx context.Context, request *pb.GetImageRequest) (*pb.GetImageResponse, error) {
	result := s.bot.CQGetImage(request.File)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetImageResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) DownloadFile(ctx context.Context, request *pb.DownloadFileRequest) (*pb.DownloadFileResponse, error) {
	result := s.bot.CQDownloadFile(request.Url, request.Headers, int(request.ThreadCount))
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.DownloadFileResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetForwardMessage(ctx context.Context, request *pb.GetForwardMessageRequest) (*pb.GetForwardMessageResponse, error) {
	result := s.bot.CQGetForwardMessage(request.MessageId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetForwardMessageResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetMessage(ctx context.Context, request *pb.GetMessageRequest) (*pb.GetMessageResponse, error) {
	result := s.bot.CQGetMessage(request.MessageId)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetMessageResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupSystemMessage(ctx context.Context, request *pb.GetGroupSystemMessageRequest) (*pb.GetGroupSystemMessageResponse, error) {
	result := s.bot.CQGetGroupSystemMessages()
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(*client.GroupSystemMessages)
	response := &pb.GetGroupSystemMessageResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetGroupMessageHistory(ctx context.Context, request *pb.GetGroupMessageHistoryRequest) (*pb.GetGroupMessageHistoryResponse, error) {
	result := s.bot.CQGetGroupMessageHistory(request.GroupId, request.MessageSeq)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetGroupMessageHistoryResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetOnlineClients(ctx context.Context, request *pb.GetOnlineClientsRequest) (*pb.GetOnlineClientsResponse, error) {
	result := s.bot.CQGetOnlineClients(request.NoCache)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(coolq.MSG)
	response := &pb.GetOnlineClientsResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) CanSendImage(ctx context.Context, request *pb.CanSendImageRequest) (*pb.CanSendImageResponse, error) {
	data := s.bot.CQCanSendImage()["data"].(coolq.MSG)
	response := &pb.CanSendImageResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) CanSendRecord(ctx context.Context, request *pb.CanSendRecordRequest) (*pb.CanSendRecordResponse, error) {
	data := s.bot.CQCanSendRecord()["data"].(coolq.MSG)
	response := &pb.CanSendRecordResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) OcrImage(ctx context.Context, request *pb.OcrImageRequest) (*pb.OcrImageResponse, error) {
	result := s.bot.CQOcrImage(request.Image)
	if err := checkError(result); err != nil {
		return nil, err
	}

	data := result["data"].(*client.OcrResponse)
	response := &pb.OcrImageResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) ReloadEventFilter(ctx context.Context, request *pb.ReloadEventFilterRequest) (*pb.ReloadEventFilterResponse, error) {
	s.bot.CQReloadEventFilter()

	return &pb.ReloadEventFilterResponse{}, nil
}

func (s *APIService) SetGroupPortrait(ctx context.Context, request *pb.SetGroupPortraitRequest) (*pb.SetGroupPortraitResponse, error) {
	result := s.bot.CQSetGroupPortrait(request.GroupId, request.File, request.Cache)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupPortraitResponse{}, nil
}

func (s *APIService) SetGroupAnonymousBan(ctx context.Context, request *pb.SetGroupAnonymousBanRequest) (*pb.SetGroupAnonymousBanResponse, error) {
	result := s.bot.CQSetGroupAnonymousBan(request.GroupId, request.Flag, request.Duration)
	if err := checkError(result); err != nil {
		return nil, err
	}

	return &pb.SetGroupAnonymousBanResponse{}, nil
}

func (s *APIService) GetStatus(ctx context.Context, request *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	result := s.bot.CQGetStatus()

	data := result["data"].(coolq.MSG)
	response := &pb.GetStatusResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetVersionInfo(ctx context.Context, request *pb.GetVersionInfoRequest) (*pb.GetVersionInfoResponse, error) {
	result := s.bot.CQGetVersionInfo()

	data := result["data"].(coolq.MSG)
	response := &pb.GetVersionInfoResponse{}
	if err := fillStruct(data, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *APIService) GetEvents(request *pb.GetEventsResquest, stream pb.API_GetEventsServer) error {
	recvChan := make(chan *pb.Event)
	s.EventPusher.Registry(recvChan)
	defer s.EventPusher.UnRegistry(recvChan)

	for {
		if err := stream.Send(<-recvChan); err != nil {
			log.Fatal(err)
		}
	}
}

func checkError(result coolq.MSG) error {
	if result["data"] == nil && result["retcode"] != 0 {
		err := &CQError{}
		if err := fillStruct(result, err); err != nil {
			return err
		}
		return err
	}

	return nil
}

func fillStruct(in interface{}, out interface{}) error {
	config := &mapstructure.DecoderConfig{TagName: "json", Result: out}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	err = decoder.Decode(in)
	if err != nil {
		return err
	}

	return nil
}
