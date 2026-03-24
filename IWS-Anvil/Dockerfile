FROM ghcr.io/foundry-rs/foundry:latest
COPY contracts/ /contracts/
COPY entrypoint.sh /entrypoint.sh
EXPOSE 8545
ENTRYPOINT ["sh", "/entrypoint.sh"]
