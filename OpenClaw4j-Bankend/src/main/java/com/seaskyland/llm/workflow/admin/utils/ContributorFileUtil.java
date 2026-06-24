package com.seaskyland.llm.workflow.admin.utils;


import io.spring.initializr.generator.project.ProjectDescription;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;

/**
 * 用于生成项目时，创建文件
 *
 * @author vlsmb
 * @since 2025/9/5
 */
public final class ContributorFileUtil {

    private ContributorFileUtil() {

    }

    public static void createResourceDirectory(Path projectRoot) {
        Path resourceDirectory = projectRoot.resolve("src/main/resources");
        if (!Files.exists(resourceDirectory)) {
            try {
                Files.createDirectories(resourceDirectory);
            }
            catch (IOException e) {
                throw new RuntimeException(e);
            }
        }
    }

    public static void saveResourceFile(Path projectRoot, String fileName, InputStream inputStream) {
        createResourceDirectory(projectRoot);
        Path resourceFile = projectRoot.resolve("src/main/resources").resolve(fileName);
        try {
            Files.copy(inputStream, resourceFile);
        }
        catch (IOException e) {
            throw new RuntimeException(e);
        }
    }

    public static Path createDirectory(Path projectRoot, ProjectDescription projectDescription) {
        StringBuilder pathBuilder = new StringBuilder("src/main/").append(projectDescription.getLanguage().id());
        String packagePath = projectDescription.getPackageName().replace('.', '/');
        pathBuilder.append("/").append(packagePath).append("/graph/");
        Path fileRoot;
        try {
            fileRoot = Files.createDirectories(projectRoot.resolve(pathBuilder.toString()));
        }
        catch (Exception e) {
            throw new RuntimeException("Got error when creating files", e);
        }
        return fileRoot;
    }

}

