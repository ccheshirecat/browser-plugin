# Browser runtime OCI image
FROM chromedp/headless-shell:stable

ARG AGENT_BINARY=browser-agent.bin

COPY ${AGENT_BINARY} /usr/local/bin/browser-agent
COPY rootfs/ /overlay/

RUN chmod +x /usr/local/bin/browser-agent

ENTRYPOINT ["/usr/local/bin/browser-agent"]
