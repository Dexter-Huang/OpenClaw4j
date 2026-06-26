FROM maven:3.9-eclipse-temurin-26 AS build

WORKDIR /workspace
COPY pom.xml .
COPY src ./src

RUN mvn -DskipTests -Pleyden package

FROM eclipse-temurin:26-jre AS runtime

WORKDIR /app
COPY --from=build /workspace/target/OpenClaw4j-Bankend.jar app.jar

ENV SPRING_PROFILES_ACTIVE=dev
ENV JAVA_OPTS="--enable-final-field-mutation=ALL-UNNAMED"
ENV OPENCLAW4J_LEYDEN_OPTS="-XX:+UnlockExperimentalVMOptions -XX:+UseCompactObjectHeaders"

RUN set -eux; \
    mkdir -p runtime; \
    cp app.jar runtime/OpenClaw4j-Bankend.jar; \
    cd runtime; \
    ls -lh OpenClaw4j-Bankend.jar; \
    java -version; \
    if ! java ${OPENCLAW4J_LEYDEN_OPTS} \
      --enable-final-field-mutation=ALL-UNNAMED \
      -XX:AOTCacheOutput=openclaw4j.aot \
      -Dspring.profiles.active=${SPRING_PROFILES_ACTIVE} \
      -Dopenclaw4j.leyden.training.profiled=true \
      -jar OpenClaw4j-Bankend.jar \
      --server.port=0 > leyden-training.log 2>&1; then \
      cat leyden-training.log; \
      exit 1; \
    fi; \
    cat leyden-training.log; \
    test -f openclaw4j.aot; \
    ls -lh openclaw4j.aot

WORKDIR /app/runtime

EXPOSE 9004

ENTRYPOINT ["sh", "-c", "exec java ${OPENCLAW4J_LEYDEN_OPTS} -XX:AOTCache=openclaw4j.aot ${JAVA_OPTS} -Dspring.profiles.active=${SPRING_PROFILES_ACTIVE:-dev} -jar OpenClaw4j-Bankend.jar"]
