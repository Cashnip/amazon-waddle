FROM node:24.20.0-alpine AS construcao
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:24.20.0-alpine
WORKDIR /app
ENV NODE_ENV=production
# `next start` lê o next.config.ts em tempo de execução — sem ele o
# rewrites() de /api some e o navegador volta a ver duas origens (AD-10).
COPY --from=construcao /app/package.json /app/package-lock.json /app/next.config.ts ./
COPY --from=construcao /app/node_modules ./node_modules
COPY --from=construcao /app/.next ./.next
COPY --from=construcao /app/public ./public
EXPOSE 3000
CMD ["npm", "run", "start"]
