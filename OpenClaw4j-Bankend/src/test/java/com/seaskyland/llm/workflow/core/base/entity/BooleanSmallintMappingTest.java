package com.seaskyland.llm.workflow.core.base.entity;

import static org.assertj.core.api.Assertions.assertThat;

import com.baomidou.mybatisplus.annotation.TableField;
import java.lang.reflect.Field;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.util.stream.Stream;
import org.apache.ibatis.type.BaseTypeHandler;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.Arguments;
import org.junit.jupiter.params.provider.MethodSource;
import org.mockito.Mockito;

class BooleanSmallintMappingTest {

  @ParameterizedTest
  @MethodSource("smallintBooleanFields")
  void smallintBooleanFieldsUseExplicitTypeHandler(Class<?> entityType, String fieldName)
      throws NoSuchFieldException {
    Field field = entityType.getDeclaredField(fieldName);
    TableField tableField = field.getAnnotation(TableField.class);

    assertThat(tableField).isNotNull();
    assertThat(tableField.typeHandler().getName())
        .isEqualTo("com.seaskyland.llm.workflow.core.base.typehandler.BooleanSmallintTypeHandler");
  }

  private static Stream<Arguments> smallintBooleanFields() {
    return Stream.of(
        Arguments.of(AgentSchemaEntity.class, "enabled"),
        Arguments.of(DocumentEntity.class, "enabled"),
        Arguments.of(ModelEntity.class, "enable"),
        Arguments.of(ProviderEntity.class, "enable"),
        Arguments.of(ToolEntity.class, "enabled"));
  }

  @Test
  void booleanSmallintTypeHandlerStoresBooleanAsNumericFlag() throws Exception {
    BaseTypeHandler<Boolean> handler = booleanSmallintTypeHandler();
    PreparedStatement statement = Mockito.mock(PreparedStatement.class);

    handler.setNonNullParameter(statement, 1, true, null);
    handler.setNonNullParameter(statement, 2, false, null);

    Mockito.verify(statement).setShort(1, (short) 1);
    Mockito.verify(statement).setShort(2, (short) 0);
  }

  @Test
  void booleanSmallintTypeHandlerReadsNumericFlagAsBoolean() throws Exception {
    BaseTypeHandler<Boolean> handler = booleanSmallintTypeHandler();
    ResultSet resultSet = Mockito.mock(ResultSet.class);
    Mockito.when(resultSet.getShort("enabled")).thenReturn((short) 1, (short) 0);
    Mockito.when(resultSet.wasNull()).thenReturn(false);

    assertThat(handler.getNullableResult(resultSet, "enabled")).isTrue();
    assertThat(handler.getNullableResult(resultSet, "enabled")).isFalse();
  }

  @SuppressWarnings("unchecked")
  private BaseTypeHandler<Boolean> booleanSmallintTypeHandler() throws Exception {
    return (BaseTypeHandler<Boolean>)
        Class.forName(
                "com.seaskyland.llm.workflow.core.base.typehandler.BooleanSmallintTypeHandler")
            .getDeclaredConstructor()
            .newInstance();
  }
}
