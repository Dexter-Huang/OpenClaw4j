FROM maven:3.9-eclipse-temurin-25 AS build

WORKDIR /workspace
COPY pom.xml .
COPY src ./src

RUN mvn -DskipTests -Pleyden package

FROM eclipse-temurin:25-jre AS runtime

WORKDIR /app
COPY --from=build /workspace/target/OpenClaw4j-Bankend.jar app.jar

RUN java -Djarmode=tools -jar app.jar extract --destination app \
    && cd app \
    && export APP_CP="OpenClaw4j-Bankend.jar:lib/*" \
    && java -XX:AOTCacheOutput=openclaw4j.aot \
      -Dspring.profiles.active=${SPRING_PROFILES_ACTIVE:-dev} \
      -Dopenclaw4j.leyden.training.profiled=true \
      -cp "$APP_CP" \
      com.seaskyland.llm.LLMApplication \
      --server.port=0 \
    && printf '%s\n' "$APP_CP" > classpath.txt

WORKDIR /app/app

EXPOSE 9004

ENTRYPOINT ["sh", "-c", "java -XX:AOTCache=openclaw4j.aot -Dspring.profiles.active=${SPRING_PROFILES_ACTIVE:-dev} -cp \"$(cat classpath.txt)\" com.seaskyland.llm.LLMApplication"]
