module github.com/carloshomar/fuudelivery/server

go 1.23.0

replace github.com/carloshomar/fuudelivery/auth-api => ../auth_api

replace github.com/carloshomar/fuudelivery/orders-api => ../orders_api

replace github.com/carloshomar/fuudelivery/delivery-api => ../delivery_api

require (
	github.com/carloshomar/fuudelivery/auth-api v0.0.0-00010101000000-000000000000
	github.com/carloshomar/fuudelivery/delivery-api v0.0.0-00010101000000-000000000000
	github.com/carloshomar/fuudelivery/orders-api v0.0.0-00010101000000-000000000000
	github.com/gofiber/contrib/websocket v1.3.4
	github.com/gofiber/fiber/v2 v2.52.14
	github.com/joho/godotenv v1.5.1
	github.com/streadway/amqp v1.1.0
)

require (
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/fasthttp/websocket v1.5.12 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240916144458-20a13a1f6b7c // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/savsgio/gotils v0.0.0-20250408102913-196191ec6287 // indirect
	github.com/tinylib/msgp v1.2.5 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.61.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
	gorm.io/gorm v1.31.2 // indirect
)
