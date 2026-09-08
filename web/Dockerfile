FROM node:24.20.0-alpine AS construcao
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:24.20.0-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=construcao /app/package.json /app/package-lock.json ./
COPY --from=construcao /app/node_modules ./node_modules
COPY --from=construcao /app/.next ./.next
COPY --from=construcao /app/public ./public
EXPOSE 3000
CMD ["npm", "run", "start"]
