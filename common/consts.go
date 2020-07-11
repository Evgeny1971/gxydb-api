package common

const GatewayTypeRooms = "rooms"
const GatewayTypeStreaming = "streaming"

const RoleGuest = "gxy_guest"
const RoleUser = "gxy_user"
const RoleShidur = "gxy_shidur"
const RoleSoundMan = "gxy_sndman"
const RoleViewer = "gxy_viewer"
const RoleAdmin = "gxy_admin"
const RoleRoot = "gxy_root"

var AllRoles = []string{RoleGuest, RoleUser, RoleShidur, RoleSoundMan, RoleViewer, RoleAdmin, RoleRoot}

const EventGatewayTokensChanged = "GATEWAY_TOKENS_CHANGED"

const APIDefaultPageSize = 50
const APIMaxPageSize = 1000
