FROM openresty/openresty:1.25.3.1-0-alpine-fat
WORKDIR /opt/smoke
COPY lua /opt/smoke/lua
COPY run_scenarios.sh /opt/smoke/run_scenarios.sh
RUN chmod +x /opt/smoke/run_scenarios.sh
CMD ["/opt/smoke/run_scenarios.sh"]
