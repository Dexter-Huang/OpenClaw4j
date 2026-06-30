package com.seaskyland.llm.workflow.core.base.manager;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.lang.reflect.Method;
import java.net.URI;
import java.net.URISyntaxException;
import java.net.URL;
import java.net.URLClassLoader;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import javax.tools.*;

/** 基于内存编译的代码执行器 不需要创建临时文件，直接在内存中编译和执行代码 */
public class InMemoryCodeExecutor {

  /**
   * 在内存中编译并执行代码中的Main类的main方法
   *
   * @param code 包含Main类的Java代码
   * @param params 输入参数
   * @return Main.main()方法的返回结果
   * @throws Exception 执行过程中可能抛出的异常
   */
  public static Map<String, Object> execute(String code, Map<String, Object> params)
      throws Exception {
    // 获取当前应用的类路径
    String classpath = System.getProperty("java.class.path");

    // 创建Java编译器
    JavaCompiler compiler = ToolProvider.getSystemJavaCompiler();
    StandardJavaFileManager standardFileManager = compiler.getStandardFileManager(null, null, null);

    // 创建内存文件管理器
    MemoryJavaFileManager fileManager = new MemoryJavaFileManager(standardFileManager);

    // 构造编译任务
    List<String> options = Arrays.asList("-cp", classpath);
    JavaFileObject file = new MemoryJavaFileObject("Main", code);

    // 执行编译
    JavaCompiler.CompilationTask task =
        compiler.getTask(null, fileManager, null, options, null, Arrays.asList(file));
    Boolean result = task.call();

    if (result == null || !result) {
      throw new RuntimeException("代码编译失败");
    }

    // 获取应用程序类路径URL
    URL[] urls = getClassPathURLs();

    // 加载编译后的类
    MemoryClassLoader classLoader = new MemoryClassLoader(fileManager.getCompiledClasses(), urls);
    Class<?> mainClass = classLoader.loadClass("Main");

    // 获取main方法
    Method mainMethod = mainClass.getMethod("main", Map.class);

    // 调用main方法并获取结果
    @SuppressWarnings("unchecked")
    Map<String, Object> methodResult =
        (Map<String, Object>) mainMethod.invoke(null, params.get("params"));

    return methodResult;
  }

  /**
   * 编译Java代码为字节码
   *
   * @param code Java代码
   * @return 字节码
   * @throws Exception 编译异常
   */
  public static byte[] compileCodeToBytecode(String code) throws Exception {
    // 获取当前应用的类路径
    String classpath = System.getProperty("java.class.path");

    // 创建Java编译器
    JavaCompiler compiler = ToolProvider.getSystemJavaCompiler();
    StandardJavaFileManager standardFileManager = compiler.getStandardFileManager(null, null, null);

    // 创建内存文件管理器
    MemoryJavaFileManager fileManager = new MemoryJavaFileManager(standardFileManager);

    // 构造编译任务
    List<String> options = Arrays.asList("-cp", classpath);
    JavaFileObject file = new MemoryJavaFileObject("Main", code);

    // 执行编译
    JavaCompiler.CompilationTask task =
        compiler.getTask(null, fileManager, null, options, null, Arrays.asList(file));
    Boolean result = task.call();

    if (result == null || !result) {
      throw new RuntimeException("代码编译失败");
    }

    // 获取编译后的字节码
    MemoryJavaClassFileObject compiledClass = fileManager.getCompiledClasses().get("Main");
    if (compiledClass == null) {
      throw new RuntimeException("编译后的类未找到");
    }

    return compiledClass.getBytes();
  }

  /**
   * 获取类路径URL数组
   *
   * @return 类路径URL数组
   */
  private static URL[] getClassPathURLs() {
    String classpath = System.getProperty("java.class.path");
    String[] classpathEntries = classpath.split(System.getProperty("path.separator"));
    URL[] urls = new URL[classpathEntries.length];
    for (int i = 0; i < classpathEntries.length; i++) {
      try {
        urls[i] = new java.io.File(classpathEntries[i]).toURI().toURL();
      } catch (Exception e) {
        throw new RuntimeException("无法转换类路径为URL: " + classpathEntries[i], e);
      }
    }
    return urls;
  }

  /** 内存中的Java文件对象 */
  static class MemoryJavaFileObject extends SimpleJavaFileObject {
    private final String code;

    public MemoryJavaFileObject(String name, String code) {
      super(URI.create("string:///" + name.replace('.', '/') + Kind.SOURCE.extension), Kind.SOURCE);
      this.code = code;
    }

    @Override
    public CharSequence getCharContent(boolean ignoreEncodingErrors) {
      return code;
    }
  }

  /** 内存中的类文件对象 */
  static class MemoryJavaClassFileObject extends SimpleJavaFileObject {
    private final ByteArrayOutputStream outputStream = new ByteArrayOutputStream();

    public MemoryJavaClassFileObject(String name) throws URISyntaxException {
      super(URI.create("byte:///" + name.replace('.', '/') + Kind.CLASS.extension), Kind.CLASS);
    }

    @Override
    public OutputStream openOutputStream() {
      return outputStream;
    }

    public byte[] getBytes() {
      return outputStream.toByteArray();
    }
  }

  /** 内存文件管理器 */
  static class MemoryJavaFileManager extends ForwardingJavaFileManager<JavaFileManager> {
    private final Map<String, MemoryJavaClassFileObject> compiledClasses =
        new java.util.HashMap<>();

    public MemoryJavaFileManager(JavaFileManager fileManager) {
      super(fileManager);
    }

    @Override
    public JavaFileObject getJavaFileForOutput(
        Location location, String className, JavaFileObject.Kind kind, FileObject sibling)
        throws IOException {
      try {
        MemoryJavaClassFileObject file = new MemoryJavaClassFileObject(className);
        compiledClasses.put(className, file);
        return file;
      } catch (URISyntaxException e) {
        throw new IOException(e);
      }
    }

    public Map<String, MemoryJavaClassFileObject> getCompiledClasses() {
      return compiledClasses;
    }
  }

  /** 内存类加载器 */
  static class MemoryClassLoader extends URLClassLoader {
    private final Map<String, MemoryJavaClassFileObject> compiledClasses;

    public MemoryClassLoader(Map<String, MemoryJavaClassFileObject> compiledClasses, URL[] urls) {
      super(urls);
      this.compiledClasses = compiledClasses;
    }

    @Override
    protected Class<?> findClass(String name) throws ClassNotFoundException {
      MemoryJavaClassFileObject file = compiledClasses.get(name);
      if (file != null) {
        byte[] bytes = file.getBytes();
        return defineClass(name, bytes, 0, bytes.length);
      }
      return super.findClass(name);
    }
  }
}
