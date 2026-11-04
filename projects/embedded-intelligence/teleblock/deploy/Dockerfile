# syntax=docker/dockerfile:1

# Image multi-arch : l'étage de build tourne sur la plateforme de l'hôte
# ($BUILDPLATFORM) et cross-compile vers $TARGETOS/$TARGETARCH — amd64
# comme arm64 (Raspberry Pi) sans émulation QEMU. Version de Go épinglée
# sur celle de go.mod (revue : un tag flottant casse la reproductibilité).
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY . .

# Binaire statique (CGO désactivé) : l'étage runtime n'embarque que
# l'outil iptables exigé par la redirection TCP (proxy SNI + reset, T1).
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
	go build -trimpath -ldflags="-s -w" -o /out/teleblock .

FROM alpine:3.20
# EXPOSE documents the full network surface read-only; under host mode
# (required for iptables REDIRECT) the ports are the host's network.
EXPOSE 53/udp 8449/tcp 8080/tcp

RUN apk add --no-cache iptables

COPY --from=build /out/teleblock /usr/local/bin/teleblock
COPY deploy/entrypoint.sh /usr/local/bin/teleblock-entrypoint
RUN chmod +x /usr/local/bin/teleblock-entrypoint

# DNS (UDP pour les requêtes, TCP pour les réponses tronquées et le
# proxy SNI) ; l'état d'apprentissage (T3) vit dans /var/lib/teleblock.
EXPOSE 53/udp 53/tcp
VOLUME /var/lib/teleblock

ENTRYPOINT ["/usr/local/bin/teleblock-entrypoint"]
