FROM openjdk:17-jdk

# 将target目录下的jar文件复制到镜像中，并重命名为ARG中定义的变量
ADD ./ces-llm-1.0.0.jar ces-llm.jar

# 运行jar文件
CMD ["java", "-jar", "ces-llm.jar"]

# 暴露端口
EXPOSE 28081
