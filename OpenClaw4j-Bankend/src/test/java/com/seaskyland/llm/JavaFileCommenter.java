package com.seaskyland.llm;

import java.io.*;
import java.nio.file.*;
import java.util.List;
import java.util.stream.Stream;

public class JavaFileCommenter {

  public static void main(String[] args) {

    String directoryPath =
        "C:\\Users\\hyt_c\\Desktop\\CBES\\ces-llm\\src\\main\\java\\com\\seaskyland\\llm\\workflow";

    File directory = new File(directoryPath);

    if (!directory.exists()) {
      System.out.println("Directory does not exist: " + directoryPath);
      return;
    }

    if (!directory.isDirectory()) {
      System.out.println("Path is not a directory: " + directoryPath);
      return;
    }

    try {
      //            commentJavaFiles(directoryPath);
      //             注意找到SAA include，去掉注释，恢复SAA功能
      uncommentJavaFiles(directoryPath);
    } catch (IOException e) {
      System.err.println("Error processing Java files: " + e.getMessage());
      e.printStackTrace();
    }
  }

  public static List<String> excludeFileName = List.of("JacksonConfig.java");

  /** Comments all Java files in the given directory by adding // at the beginning of each line */
  public static void commentJavaFiles(String directoryPath) throws IOException {
    try (Stream<Path> paths = Files.walk(Paths.get(directoryPath))) {
      paths
          .filter(Files::isRegularFile)
          .filter(
              path ->
                  path.toString().endsWith(".java")
                      && !path.toString().contains("JacksonConfig.java"))
          .forEach(JavaFileCommenter::commentFile);
    }
  }

  private static void commentFile(Path filePath) {
    try {
      // Read all lines
      String content = Files.readString(filePath);
      String[] lines = content.split("\n", -1);

      // Add // to the beginning of each line
      StringBuilder commentedContent = new StringBuilder();
      for (int i = 0; i < lines.length; i++) {
        commentedContent.append("//").append(lines[i]);
        if (i < lines.length - 1) {
          commentedContent.append("\n");
        }
      }

      // Write back to file
      Files.writeString(filePath, commentedContent.toString());
      System.out.println("Commented: " + filePath);
    } catch (IOException e) {
      System.err.println("Error commenting file " + filePath + ": " + e.getMessage());
    }
  }

  /** Removes // from the beginning of each line in all Java files in the given directory */
  public static void uncommentJavaFiles(String directoryPath) throws IOException {
    try (Stream<Path> paths = Files.walk(Paths.get(directoryPath))) {
      paths
          .filter(Files::isRegularFile)
          .filter(
              path ->
                  path.toString().endsWith(".java")
                      && !path.toString().contains("JacksonConfig.java"))
          .forEach(JavaFileCommenter::uncommentFile);
    }
  }

  private static void uncommentFile(Path filePath) {
    try {
      // Read all lines
      String content = Files.readString(filePath);
      String[] lines = content.split("\n", -1);

      // Remove // from the beginning of each line if present
      StringBuilder uncommentedContent = new StringBuilder();
      for (int i = 0; i < lines.length; i++) {
        if (lines[i].startsWith("//")) {
          uncommentedContent.append(lines[i].substring(2));
        } else {
          uncommentedContent.append(lines[i]);
        }
        if (i < lines.length - 1) {
          uncommentedContent.append("\n");
        }
      }

      // Write back to file
      Files.writeString(filePath, uncommentedContent.toString());
      System.out.println("Uncommented: " + filePath);
    } catch (IOException e) {
      System.err.println("Error uncommenting file " + filePath + ": " + e.getMessage());
    }
  }
}
