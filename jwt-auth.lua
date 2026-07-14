local jwt = require "resty.jwt"
local validators = require "resty.jwt-validators"


local secret_file_path = "/etc/nginx/jwt-secret-file"

-- Lê a chave secret do arquivo
local secret_file = io.open(secret_file_path, "r")
local secret = secret_file and secret_file:read("*a")
secret_file:close()

-- Remove possíveis quebras de linha ou espaços em branco extras
secret = secret and secret:gsub("%s+", "")

-- Se a chave secreta não estiver definida, retorna HTTP_INTERNAL_SERVER_ERROR
if not secret or secret == "" then
    ngx.status = ngx.HTTP_INTERNAL_SERVER_ERROR
    ngx.header.content_type = "application/json; charset=utf-8"
    ngx.say('{"error": "JWT_SECRET not set in server."}')
    ngx.exit(ngx.HTTP_INTERNAL_SERVER_ERROR)
end



-- Rotas públicas que não exigem autenticação JWT
local public_routes = {
    "login",
    "register",
    "webhook",
    "establishments",
}

-- Verifica se a URI atual corresponde a alguma rota pública
local function is_public_route(uri)
    for _, route in ipairs(public_routes) do
        if string.match(uri, route) then
            return true
        end
    end
    return false
end

-- Pula validação para OPTIONS (CORS preflight) e rotas públicas
if ngx.var.request_method ~= "OPTIONS" and not is_public_route(ngx.var.uri) then
    local authHeader = ngx.var.http_Authorization

    if authHeader == nil then
        ngx.status = ngx.HTTP_UNAUTHORIZED
        ngx.header.content_type = "application/json; charset=utf-8"
        ngx.say('{"error": "Token JWT não fornecido"}')
        ngx.exit(ngx.HTTP_UNAUTHORIZED)
    end

    -- Remove prefixo "Bearer " se presente
    local jwtToken = authHeader
    local _, _, bearerToken = string.find(authHeader, "Bearer%s+(.+)")
    if bearerToken then
        jwtToken = bearerToken
    end

    if not secret then
        ngx.status = ngx.HTTP_INTERNAL_SERVER_ERROR
        ngx.header.content_type = "application/json; charset=utf-8"
        ngx.say('{"error": "JWT_SECRET não configurado no servidor"}')
        ngx.exit(ngx.HTTP_INTERNAL_SERVER_ERROR)
    end

    local claim_spec = {
        exp = validators.is_not_expired()
    }

    local jwt_obj = jwt:load_jwt(jwtToken)
    local verified = jwt:verify_jwt_obj(secret, jwt_obj)

    if not verified then
        ngx.status = ngx.HTTP_UNAUTHORIZED
        ngx.header.content_type = "application/json; charset=utf-8"
        ngx.say('{"error": "Token JWT inválido ou expirado"}')
        ngx.exit(ngx.HTTP_UNAUTHORIZED)
    end
end
