package com.seaskyland.llm;

import java.util.regex.Pattern;
import java.util.regex.Matcher;

public class ImageTagMatcher {

    public static void main(String[] args) {
        String input1 = "This is a test <img1/>";
        String input2 = "This is another test <img2/>";
        String input3 = "This is not a match <img3>";
        String input4 = "This is not a match at all";

        System.out.println(matchesImageTag(input1)); // true
        System.out.println(matchesImageTag(input2)); // true
        System.out.println(matchesImageTag(input3)); // false
        System.out.println(matchesImageTag(input4)); // false
    }

    public static boolean matchesImageTag(String input) {
        // 正则表达式模式：以 <img[数字]/> 结尾
        String pattern = "<img\\d+/>$";
        Pattern compiledPattern = Pattern.compile(pattern);
        Matcher matcher = compiledPattern.matcher(input);
        return matcher.find();
    }
}
