package emitter

// fork from https://github.com/nsxdevx/nsxbot

type Action = string

const (
	ACTION_SEND_PRIVATE_MSG        Action = "send_private_msg"
	ACTION_SEND_GROUP_MSG          Action = "send_group_msg"
	ACTION_GET_MSG                 Action = "get_msg"
	ACTION_DELETE_MSG              Action = "delete_msg"
	ACTION_GET_LOGIN_INFO          Action = "get_login_info"
	ACTION_GET_STRANGER_INFO       Action = "get_stranger_info"
	ACTION_GET_STATUS              Action = "get_status"
	ACTION_GET_VERSION_INFO        Action = "get_version_info"
	ACTION_SET_FRIEND_ADD_REQUEST  Action = "set_friend_add_request"
	ACTION_SET_GROUP_ADD_REQUEST   Action = "set_group_add_request"
	ACTION_SET_GROUP_SPECIAL_TITLE Action = "set_group_special_title"
)

// 这些分开的原因：他们并非Onebot11大典里定义的内容。（虽然那个大典感觉参考价值存疑）
const (
	ACTION_QUIT_GROUP            Action = "set_group_leave"       // 退群
	ACTION_SET_GROUP_CARD        Action = "set_group_card"        // 设置群名片
	ACTION_GET_GROUP_INFO        Action = "get_group_info"        // 获取群信息
	ACTION_GET_GROUP_MEMBER_INFO Action = "get_group_member_info" // 获取群成员信息
)
