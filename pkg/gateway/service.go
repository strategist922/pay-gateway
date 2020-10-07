package gateway

import (
	"flag"
	"fmt"
	"github.com/pjoc-team/pay-gateway/pkg/configclient"
	"github.com/pjoc-team/pay-gateway/pkg/generator"
	"github.com/pjoc-team/pay-gateway/pkg/model"
	"github.com/pjoc-team/pay-gateway/pkg/service"
	"github.com/pjoc-team/pay-gateway/pkg/validator"
	pb "github.com/pjoc-team/pay-proto/go"
	"github.com/pjoc-team/tracing/logger"
	"golang.org/x/net/context"
)

type PayGatewayService struct {
	dbServiceClient pb.PayDatabaseServiceClient
	discovery       *service.Discovery
	configclients   *configclient.ConfigClients
	payConfig       *model.PayConfig
	orderGenerator  *generator.Generator
}

type RequestContext struct {
	GatewayOrderId     string
	ChannelAccount     string
	PayRequest         *pb.PayRequest
	PayOrder           *pb.PayOrder
	ChannelPayRequest  *pb.ChannelPayRequest
	ChannelPayResponse *pb.ChannelPayResponse
	err                error
}

func BuildParamsErrorResponse(err error) *pb.PayResponse {
	response := &pb.PayResponse{}
	response.Result = &pb.ReturnResult{Code: pb.ReturnResultCode_CODE_PARAMS_ERROR, Message: "PARAMS_ERROR", Describe: err.Error()}
	return response
}
func BuildSystemErrorResponse(err error) *pb.PayResponse {
	response := &pb.PayResponse{}
	response.Result = &pb.ReturnResult{Code: pb.ReturnResultCode_CODE_SYSTEM_ERROR, Message: "SYSTEM_ERROR", Describe: err.Error()}
	return response
}

func (svc *PayGatewayService) Pay(ctx context.Context, request *pb.PayRequest) (response *pb.PayResponse, err error) {
	log := logger.ContextLog(ctx)
	log.Debugf("New request: %v", request)
	if err = request.Validate(); err != nil {
		return BuildParamsErrorResponse(err), nil
	}
	if err = validator.Validate(ctx, *request, svc.configclients.GetAppConfig); err != nil {
		return BuildParamsErrorResponse(err), nil
	}
	var cfg *model.AppIdChannelConfig
	if cfg, err = svc.processChannelIdIfNotPresent(ctx, request); err != nil {
		err = fmt.Errorf("could'nt found config of channelId: %v", request.ChannelId)
		return BuildParamsErrorResponse(err), nil
	}

	response = &pb.PayResponse{}
	requestContext := &RequestContext{}

	gatewayOrderId := svc.orderGenerator.GenerateId()
	requestContext.GatewayOrderId = gatewayOrderId
	requestContext.PayRequest = request
	requestContext.ChannelAccount = cfg.ChannelAccount

	if result, e := svc.SavePayOrder(ctx, requestContext); e != nil || result.Code != pb.ReturnResultCode_CODE_SUCCESS {
		err = e
		response.Result = result
		return BuildSystemErrorResponse(err), nil
	}

	var client pb.PayChannelClient
	client, err = svc.discovery.GetChannelClient(request.GetChannelId())
	if client == nil || err != nil {
		log.Errorf("Failed to get channelClient! channelId: %s, error: %s, ", request.GetChannelId(), err.Error())
		return BuildSystemErrorResponse(err), nil
	} else {
		log.Debugf("Got client: %v for channelId: %s", client, request.GetChannelId())
	}
	var channelPayRequest *pb.ChannelPayRequest
	if channelPayRequest, err = svc.GenerateChannelPayRequest(ctx, requestContext); err != nil {
		return BuildSystemErrorResponse(err), nil
	}
	var channelPayResponse *pb.ChannelPayResponse
	if channelPayResponse, err = client.Pay(ctx, channelPayRequest); err != nil {
		log.Errorf("Pay channel failed! err: %s channelPayResponse: %v", err.Error(), channelPayResponse)
		requestContext.err = err
		order, err := svc.UpdatePayOrder(ctx, requestContext)
		if err != nil {
			log.Errorf("failed to update pay order: %#v error: %v", order, err.Error())
		}
		return BuildSystemErrorResponse(err), nil
	} else if channelPayResponse == nil || channelPayResponse.Data == nil {
		err = fmt.Errorf("channel response fail! response: %v", channelPayResponse)
		log.Errorf(err.Error())
		return BuildSystemErrorResponse(err), nil
	}
	requestContext.ChannelPayResponse = channelPayResponse
	if result, e := svc.UpdatePayOrder(ctx, requestContext); e != nil || result.Code != pb.ReturnResultCode_CODE_SUCCESS {
		response.Result = result
		err = e
		return
	}

	response.Data = channelPayResponse.Data
	response.GatewayOrderId = gatewayOrderId
	response.Result = SUCCESS_RESULT

	return response, nil
}

// 如果没有传入channelId，则根据method找可用的channelId
func (svc *PayGatewayService) processChannelIdIfNotPresent(ctx context.Context, request *pb.PayRequest) (channelConfig *model.AppIdChannelConfig, err error) {
	log := logger.ContextLog(ctx)

	channelConfig, err = svc.configclients.GetAppChannelConfig(request.AppId, request.Method.String())
	//if config.ChannelConfigs == nil {
	//	err = fmt.Errorf("failed to found info of appId: %v", request.AppId)
	//	return
	//}
	//for _, config := range config.ChannelConfigs {
	//	found := (request.ChannelId != "" && request.ChannelId == config.ChannelId && config.Available) ||
	//		(request.ChannelId == "" && config.Method == request.Method && config.Available)
	//
	//	if found {
	//		channelConfig = config
	//		log.Infof("find config: %v by request: %v", config, request)
	//		return
	//	}
	//}
	if err != nil {
		err = fmt.Errorf("could'nt found available channel of appId: %v method: %v", request.AppId, request.Method)
		log.Errorf(err.Error())
	}
	return
}

func NewPayGateway(cc *configclient.ConfigClients, clusterID string, concurrency int, dbServiceClient pb.PayDatabaseServiceClient) (pb.PayGatewayServer, error) {
	flag.Parse()
	payGatewayService := &PayGatewayService{}
	payGatewayService.configclients = cc
	payGatewayService.orderGenerator = generator.New(clusterID, concurrency)
	payGatewayService.dbServiceClient = dbServiceClient
	return payGatewayService, nil
}
