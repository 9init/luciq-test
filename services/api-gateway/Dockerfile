FROM nginx:1.25-alpine
RUN rm /etc/nginx/nginx.conf
COPY nginx.conf /etc/nginx/nginx.conf
RUN mkdir -p /var/log/nginx
# EXPOSE 80
# Start nginx
CMD ["nginx", "-g", "daemon off;"]
