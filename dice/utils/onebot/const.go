package emitter

type Action = string

const (
	ACTION_SEND_PRIVATE_MSG        = "send_private_msg"
	ACTION_SEND_GROUP_MSG          = "send_group_msg"
	ACTION_GET_MSG                 = "get_msg"
	ACTION_DELETE_MSG              = "delete_msg"
	ACTION_GET_LOGIN_INFO          = "get_login_info"
	ACTION_GET_STRANGER_INFO       = "get_stranger_info"
	ACTION_GET_STATUS              = "get_status"
	ACTION_GET_VERSION_INFO        = "get_version_info"
	ACTION_SET_FRIEND_ADD_REQUEST  = "set_friend_add_request"
	ACTION_SET_GROUP_ADD_REQUEST   = "set_group_add_request"
	ACTION_SET_GROUP_SPECIAL_TITLE = "set_group_special_title"
)

const (
	ACTION_QUIT_GROUP = "set_group_leave" // 退群
)
