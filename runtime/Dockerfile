# Browser runtime OCI image
FROM chromedp/headless-shell:stable

ARG AGENT_BINARY=runtime/browser-agent.bin

COPY ${AGENT_BINARY} /usr/local/bin/browser-agent
COPY runtime/rootfs/ /overlay/

RUN chmod +x /usr/local/bin/browser-agent

ENTRYPOINT ["/usr/local/bin/browser-agent"]
