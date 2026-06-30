package com.seaskyland.llm.workflow.core.config;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.seaskyland.llm.workflow.core.base.entity.AccountEntity;
import com.seaskyland.llm.workflow.core.base.entity.AppEntity;
import com.seaskyland.llm.workflow.core.base.mapper.DocumentMapper;
import org.junit.jupiter.api.Test;
import org.springframework.aot.hint.MemberCategory;
import org.springframework.aot.hint.RuntimeHints;
import org.springframework.aot.hint.predicate.RuntimeHintsPredicates;

class MybatisPlusRuntimeHintsTest {

  @Test
  void registersMybatisPlusReflectionAndResourceHints() {
    RuntimeHints hints = new RuntimeHints();

    new MybatisPlusRuntimeHints().registerHints(hints, getClass().getClassLoader());

    assertTrue(
        RuntimeHintsPredicates.reflection()
            .onType(AccountEntity.class)
            .withMemberCategories(
                MemberCategory.INVOKE_DECLARED_CONSTRUCTORS,
                MemberCategory.INVOKE_PUBLIC_METHODS,
                MemberCategory.ACCESS_DECLARED_FIELDS)
            .test(hints));
    assertTrue(
        RuntimeHintsPredicates.reflection()
            .onType(AppEntity.class)
            .withMemberCategories(
                MemberCategory.INVOKE_DECLARED_CONSTRUCTORS,
                MemberCategory.INVOKE_PUBLIC_METHODS,
                MemberCategory.ACCESS_DECLARED_FIELDS)
            .test(hints));
    assertTrue(RuntimeHintsPredicates.reflection().onType(DocumentMapper.class).test(hints));
    assertTrue(
        RuntimeHintsPredicates.resource().forResource("mapper/DocumentMapper.xml").test(hints));
    assertEquals(16, MybatisPlusRuntimeHints.MYBATIS_PLUS_ENTITY_TYPES.size());
    assertEquals(15, MybatisPlusRuntimeHints.MYBATIS_PLUS_MAPPER_TYPES.size());
  }
}
