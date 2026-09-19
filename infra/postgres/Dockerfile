# Postgres com o script de criação do papel da aplicação embutido: a imagem
# não depende de montar arquivo do repositório (o que nem todo host permite).
FROM postgres:17-alpine
COPY init.sh /docker-entrypoint-initdb.d/00-init.sh
