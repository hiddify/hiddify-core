package daemon

import (
	"context"
	"io"
	"net/url"
	"strings"
	"sync"

	controlv1 "github.com/hiddify/hiddify-core/v2/controlv1"
	hcore "github.com/hiddify/hiddify-core/v2/hcore"
	profiles "github.com/hiddify/hiddify-core/v2/profile"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const controlAPIMajor = 1
const maxLocalProfileBytes = 8 << 20
const eventQueueSize = 32

// ControlServer serves the local control.v1 API. It tracks connection and
// profile state derived from the core; requests beyond the implemented set use
// the generated Unimplemented response rather than silently reporting success.
type ControlServer struct {
	controlv1.UnimplementedControlServiceServer

	mu       sync.RWMutex
	snapshot *controlv1.Snapshot
	watchers map[chan *controlv1.Event]struct{}
	socket   string
}

func NewControlServer() *ControlServer {
	return &ControlServer{
		watchers: make(map[chan *controlv1.Event]struct{}),
		snapshot: &controlv1.Snapshot{
			ApiMajor:        controlAPIMajor,
			ConnectionState: controlv1.ConnectionState_CONNECTION_STATE_STOPPED,
			Capabilities:    []string{"daemon-lifecycle", "profiles"},
		},
	}
}

func (s *ControlServer) GetSnapshot(context.Context, *controlv1.GetSnapshotRequest) (*controlv1.Snapshot, error) {
	s.refreshActiveProfile()
	s.mu.RLock()
	defer s.mu.RUnlock()
	return proto.Clone(s.snapshot).(*controlv1.Snapshot), nil
}

// WatchEvents streams revisioned changes after a client snapshot. A slow
// subscriber is disconnected rather than allowed to block daemon work.
func (s *ControlServer) WatchEvents(request *controlv1.WatchRequest, stream grpc.ServerStreamingServer[controlv1.Event]) error {
	updates, initial := s.subscribe(request.GetAfterSequence())
	defer s.unsubscribe(updates)
	if initial != nil {
		if err := stream.Send(initial); err != nil {
			return err
		}
	}
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if err := stream.Send(update); err != nil {
				return err
			}
		}
	}
}

func (s *ControlServer) Connect(ctx context.Context, request *controlv1.ConnectRequest) (*controlv1.OperationResult, error) {
	entity, err := profiles.GetById(request.GetProfileId())
	if err != nil {
		return operation(controlv1.ErrorCode_ERROR_CODE_NOT_FOUND, "profile not found"), nil
	}
	mode := controlModeName(request.GetMode())
	if err := hcore.SetConnectionMode(mode); err != nil {
		return operation(controlv1.ErrorCode_ERROR_CODE_INTERNAL, "configure connection mode"), nil
	}
	s.setConnection(controlv1.ConnectionState_CONNECTION_STATE_STARTING, mode, entity.GetName(), entity.GetId(), "")
	response, startErr := hcore.Start(ctx, &hcore.StartRequest{ConfigPath: profileConfigPath(entity), ConfigName: entity.GetName()})
	if startErr != nil || response.GetCoreState() != hcore.CoreStates_STARTED {
		if response.GetMessageType() == hcore.MessageType_ALREADY_STARTED {
			s.setConnection(controlv1.ConnectionState_CONNECTION_STATE_STARTED, mode, entity.GetName(), entity.GetId(), "")
			return &controlv1.OperationResult{ErrorCode: controlv1.ErrorCode_ERROR_CODE_ALREADY_IN_REQUESTED_STATE, Message: "already connected", AlreadyInRequestedState: true}, nil
		}
		s.setConnection(controlv1.ConnectionState_CONNECTION_STATE_FAILED, mode, entity.GetName(), entity.GetId(), response.GetMessage())
		return operation(startFailureCode(startErr), "connect failed"), nil
	}
	s.setConnection(controlv1.ConnectionState_CONNECTION_STATE_STARTED, mode, entity.GetName(), entity.GetId(), "")
	return operation(controlv1.ErrorCode_ERROR_CODE_OK, "connected"), nil
}

func (s *ControlServer) Disconnect(ctx context.Context, _ *controlv1.DisconnectRequest) (*controlv1.OperationResult, error) {
	response, err := hcore.Stop()
	if err != nil && response.GetCoreState() != hcore.CoreStates_STOPPED {
		s.setConnection(controlv1.ConnectionState_CONNECTION_STATE_FAILED, "", "", "", response.GetMessage())
		return operation(controlv1.ErrorCode_ERROR_CODE_INTERNAL, "disconnect failed"), nil
	}
	s.setConnection(controlv1.ConnectionState_CONNECTION_STATE_STOPPED, "", "", "", "")
	return operation(controlv1.ErrorCode_ERROR_CODE_OK, "disconnected"), nil
}

func (s *ControlServer) Restart(ctx context.Context, _ *controlv1.RestartRequest) (*controlv1.OperationResult, error) {
	entity, err := profiles.GetActiveProfile()
	if err != nil {
		return operation(controlv1.ErrorCode_ERROR_CODE_NO_ACTIVE_PROFILE, "no active profile"), nil
	}
	response, restartErr := hcore.Restart(ctx, &hcore.StartRequest{ConfigPath: profileConfigPath(entity), ConfigName: entity.GetName()})
	if restartErr != nil || response.GetCoreState() != hcore.CoreStates_STARTED {
		return operation(startFailureCode(restartErr), "restart failed"), nil
	}
	return operation(controlv1.ErrorCode_ERROR_CODE_OK, "restarted"), nil
}

func (s *ControlServer) ListProfiles(context.Context, *controlv1.ListProfilesRequest) (*controlv1.ListProfilesResponse, error) {
	all, err := profiles.GetAll()
	if err != nil {
		return nil, status.Error(codes.Internal, "load profiles")
	}
	activeID := s.activeProfileID()
	response := &controlv1.ListProfilesResponse{Profiles: make([]*controlv1.Profile, 0, len(all))}
	for _, entity := range all {
		response.Profiles = append(response.Profiles, profileToControl(entity, entity.GetId() == activeID))
	}
	return response, nil
}

func (s *ControlServer) GetProfile(_ context.Context, request *controlv1.GetProfileRequest) (*controlv1.Profile, error) {
	if request.GetProfileId() == "" {
		return nil, status.Error(codes.InvalidArgument, "profile_id is required")
	}
	entity, err := profiles.GetById(request.GetProfileId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}
	return profileToControl(entity, entity.GetId() == s.activeProfileID()), nil
}

func (s *ControlServer) AddRemoteProfile(ctx context.Context, request *controlv1.AddRemoteProfileRequest) (*controlv1.Profile, error) {
	entity, err := profiles.AddByUrl(ctx, request.GetUrl(), request.GetName(), request.GetSetActive())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "add remote profile: %v", err)
	}
	return profileToControl(entity, request.GetSetActive()), nil
}

func (s *ControlServer) AddLocalProfile(stream grpc.ClientStreamingServer[controlv1.AddLocalProfileRequest, controlv1.Profile]) error {
	var name string
	var setActive bool
	var content strings.Builder
	for {
		request, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if metadata := request.GetMetadata(); metadata != nil {
			name = metadata.GetName()
			setActive = metadata.GetSetActive()
			continue
		}
		content.Write(request.GetContentChunk())
		if content.Len() > maxLocalProfileBytes {
			return status.Error(codes.InvalidArgument, "profile content exceeds size limit")
		}
	}
	entity, err := profiles.AddByContent(stream.Context(), content.String(), name, setActive)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "add local profile: %v", err)
	}
	return stream.SendAndClose(profileToControl(entity, setActive))
}

func (s *ControlServer) DeleteProfile(_ context.Context, request *controlv1.DeleteProfileRequest) (*controlv1.OperationResult, error) {
	if err := profiles.DeleteById(request.GetProfileId()); err != nil {
		return nil, status.Error(codes.Internal, "delete profile")
	}
	return operation(controlv1.ErrorCode_ERROR_CODE_OK, "profile deleted"), nil
}

func (s *ControlServer) SetActiveProfile(_ context.Context, request *controlv1.SetActiveProfileRequest) (*controlv1.OperationResult, error) {
	entity, err := profiles.GetById(request.GetProfileId())
	if err != nil {
		return operation(controlv1.ErrorCode_ERROR_CODE_NOT_FOUND, "profile not found"), nil
	}
	if err := profiles.SetActiveProfile(entity); err != nil {
		return operation(controlv1.ErrorCode_ERROR_CODE_INTERNAL, "activate profile"), nil
	}
	s.refreshActiveProfile()
	return operation(controlv1.ErrorCode_ERROR_CODE_OK, "active profile updated"), nil
}

// ServeControl runs control.v1 on the runtime's Unix listener until ctx ends.
func (r *Runtime) ServeControl(ctx context.Context, service *ControlServer) error {
	service.mu.Lock()
	service.socket = r.socket
	service.mu.Unlock()
	server := grpc.NewServer()
	controlv1.RegisterControlServiceServer(server, service)
	errors := make(chan error, 1)
	go func() { errors <- server.Serve(r.listener) }()
	select {
	case <-ctx.Done():
		server.Stop()
		return nil
	case err := <-errors:
		return err
	}
}

func (s *ControlServer) subscribe(after uint64) (chan *controlv1.Event, *controlv1.Event) {
	updates := make(chan *controlv1.Event, eventQueueSize)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.watchers[updates] = struct{}{}
	if after != 0 && after < s.snapshot.GetEventSequence() {
		return updates, &controlv1.Event{
			Sequence: s.snapshot.GetEventSequence(), Revision: s.snapshot.GetRevision(),
			Change: &controlv1.Event_ResyncRequired{ResyncRequired: &controlv1.ResyncRequired{}},
		}
	}
	return updates, nil
}

func (s *ControlServer) unsubscribe(updates chan *controlv1.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.watchers[updates]; ok {
		delete(s.watchers, updates)
		close(updates)
	}
}

// publishLocked emits a revisioned event; the caller must hold s.mu.
func (s *ControlServer) publishLocked(event *controlv1.Event) {
	s.snapshot.EventSequence++
	event.Sequence = s.snapshot.GetEventSequence()
	event.Revision = s.snapshot.GetRevision()
	for updates := range s.watchers {
		select {
		case updates <- event:
		default:
			delete(s.watchers, updates)
			close(updates)
		}
	}
}

func (s *ControlServer) setConnection(state controlv1.ConnectionState, mode, profileName, profileID, lastError string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.Revision++
	s.snapshot.ConnectionState = state
	s.snapshot.RequestedMode = mode
	s.snapshot.ActiveProfileName = profileName
	s.snapshot.ActiveProfileId = profileID
	s.snapshot.LastError = lastError
	s.publishLocked(&controlv1.Event{Change: &controlv1.Event_Connection{Connection: &controlv1.ConnectionChange{
		State: state, RequestedMode: mode, EffectiveMode: mode,
	}}})
}

func (s *ControlServer) refreshActiveProfile() {
	entity, err := profiles.GetActiveProfile()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil || entity == nil {
		return
	}
	s.snapshot.ActiveProfileId = entity.GetId()
	s.snapshot.ActiveProfileName = entity.GetName()
}

func (s *ControlServer) activeProfileID() string {
	s.refreshActiveProfile()
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot.GetActiveProfileId()
}

func profileConfigPath(entity *profiles.ProfileEntity) string {
	return "data/profiles/" + entity.GetId() + ".info"
}

func profileToControl(entity *profiles.ProfileEntity, active bool) *controlv1.Profile {
	kind := controlv1.ProfileKind_PROFILE_KIND_REMOTE
	if entity.GetUrl() == "" {
		kind = controlv1.ProfileKind_PROFILE_KIND_LOCAL
	}
	return &controlv1.Profile{
		Id:          entity.GetId(),
		Name:        entity.GetName(),
		Kind:        kind,
		Active:      active,
		RedactedUrl: redactURL(entity.GetUrl()),
	}
}

func redactURL(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	query := parsed.Query()
	for key := range query {
		query.Set(key, "[redacted]")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func controlModeName(mode controlv1.ConnectionMode) string {
	switch mode {
	case controlv1.ConnectionMode_CONNECTION_MODE_TUN:
		return "tun"
	case controlv1.ConnectionMode_CONNECTION_MODE_SYSTEM_PROXY:
		return "system-proxy"
	case controlv1.ConnectionMode_CONNECTION_MODE_LOCAL_PROXY:
		return "local-proxy"
	default:
		return "tun"
	}
}

func operation(code controlv1.ErrorCode, message string) *controlv1.OperationResult {
	return &controlv1.OperationResult{ErrorCode: code, Message: message}
}

func startFailureCode(err error) controlv1.ErrorCode {
	if err == nil {
		return controlv1.ErrorCode_ERROR_CODE_INTERNAL
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "address already in use"), strings.Contains(message, "bind"):
		return controlv1.ErrorCode_ERROR_CODE_PORT_CONFLICT
	case strings.Contains(message, "invalid"), strings.Contains(message, "config"):
		return controlv1.ErrorCode_ERROR_CODE_INVALID_CONFIGURATION
	default:
		return controlv1.ErrorCode_ERROR_CODE_INTERNAL
	}
}

var _ controlv1.ControlServiceServer = (*ControlServer)(nil)
