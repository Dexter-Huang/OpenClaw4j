FROM maven:3.9-eclipse-temurin-26 AS build

WORKDIR /workspace
COPY pom.xml .
COPY src ./src

RUN mvn -DskipTests -Pleyden package

FROM eclipse-temurin:26-jre AS runtime

WORKDIR /app
COPY --from=build /workspace/target/OpenClaw4j-Bankend.jar app.jar

ENV SPRING_PROFILES_ACTIVE=dev
ENV JAVA_OPTS=""
ENV OPENCLAW4J_LEYDEN_OPTS="-XX:+UnlockExperimentalVMOptions -XX:+UseCompactObjectHeaders"

RUN java -Djarmode=tools -jar app.jar extract --destination app \
    && cd app \
    && java ${OPENCLAW4J_LEYDEN_OPTS} \
      -XX:AOTCacheOutput=openclaw4j.aot \
      -Dspring.profiles.active=${SPRING_PROFILES_ACTIVE} \
      -Dopenclaw4j.leyden.training.profiled=true \
      -jar OpenClaw4j-Bankend.jar \
      --server.port=0 \
    && test -f openclaw4j.aot \
    && ls -lh openclaw4j.aot

WORKDIR /app/app

EXPOSE 9004

ENTRYPOINT ["sh", "-c", "exec java ${OPENCLAW4J_LEYDEN_OPTS} -XX:AOTCache=openclaw4j.aot ${JAVA_OPTS} -Dspring.profiles.active=${SPRING_PROFILES_ACTIVE:-dev} -jar OpenClaw4j-Bankend.jar"]
