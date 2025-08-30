package handler

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func JWTAuthInterceptor(jwtSecret string, logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn("Missing metadata in grpc request")
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md["authorization"]
		if len(authHeaders) == 0 {
			logger.Warn("Authorization token not supplied")
			return nil, status.Error(codes.Unauthenticated, "authorization token not supplied")
		}

		tokenString := authHeaders[0]
		if !strings.HasPrefix(tokenString, "Bearer ") {
			logger.Warn("Bad authorization token foramt")
			return nil, status.Error(codes.Unauthenticated, "bad authorization token foramt")
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				logger.Warn("unexpected singned methos: %v", zap.String("SingningMethod", t.Method.Alg()))
				return nil, nil
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			logger.Warn("Invalid token", zap.Error(err))
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(ctx, req)

	}
}
