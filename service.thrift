struct RegisterReq {
    1: string name,
    2: string email,
    3: string password
}

struct LoginReq {
    1: string email,
    2: string password
}

struct LoginData {
    1: i32 user_id,
    2: string avatar,
    3: string name,
    4: string email,
    5: string gender,
    6: string payment_id,
    7: bool is_admin,
    8: string create_at,
    9: string update_at,
    10: string token,
    11: string expire
}

struct LoginResp {
    1: i32 code,
    2: string message,
    3: LoginData data
}

struct UserResp {
    1: i32 code,
    2: string message,
    3: User data
}

struct UserListResp {
    1: i32 code,
    2: string message,
    3: list<User> data
}

struct SuccessResp {
    1: i32 code,
    2: string message
}

struct AdminQueryUserReq {
    1: i32 user_id
}

struct User {
    1: i32 user_id,
    2: string avatar,
    3: string name,
    4: string email,
    5: string gender,
    6: string payment_id,
    7: bool is_admin,
    8: string create_at,
    9: string update_at
}

service UserService {
    UserResp Register(1: RegisterReq req) (api.post="/api/user/register");

    UserResp Login(1: LoginReq req) (api.post="/api/user/login");

    UserListResp AdminQueryAllUsers() (api.get="/api/user");

    SuccessResp AdminDeleteUser(1: AdminQueryUserReq req) (api.delete="/api/user/:user_id");

    UserResp AdminQueryUser(1: AdminQueryUserReq req) (api.get="/api/user/:user_id");
}