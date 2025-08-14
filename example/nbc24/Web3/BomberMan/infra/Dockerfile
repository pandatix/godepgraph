FROM debian:stable-slim

RUN apt update -y && \
    apt install git curl python3 python3-pip supervisor -y && \
    python3 -m pip install eth_account flask requests web3 gunicorn --break-system-packages && \
    useradd -m -s /bin/bash user

COPY ./src/supervisor/supervisord.conf /etc/supervisor/conf.d/
COPY ./src/app/ /home/user/app/

RUN mkdir /home/user/blockchain
RUN chown -R user:user /home/user/

USER user

RUN curl -L https://foundry.paradigm.xyz | bash && \
    /home/user/.foundry/bin/foundryup && \
    echo 'alias ll="ls -lah"' >> /home/user/.bashrc

RUN cd /home/user/blockchain && \
    /home/user/.foundry/bin/forge install 'openzeppelin/openzeppelin-contracts@release-v5.0' --no-git && \
    /home/user/.foundry/bin/forge build && \
    /home/user/.foundry/bin/forge compile

RUN cd /home/user/blockchain && \
    /home/user/.foundry/bin/forge init --force --no-git && \
    rm -rf /home/user/blockchain/src/* && \
    rm -rf /home/user/blockchain/test/* && \
    rm -rf /home/user/blockchain/script/* && \
    rm -rf /home/user/blockchain/foundry.toml

COPY ./src/blockchain/ /home/user/blockchain/
    
USER root

RUN rm -rf /var/cache/apt/archives /var/lib/apt/lists/*

EXPOSE 8000

CMD ["/usr/bin/supervisord","-c","/etc/supervisor/supervisord.conf"]
