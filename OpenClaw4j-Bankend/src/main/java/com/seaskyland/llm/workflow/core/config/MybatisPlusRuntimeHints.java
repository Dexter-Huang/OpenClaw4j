package com.seaskyland.llm.workflow.core.config;

import java.util.List;

import com.seaskyland.llm.workflow.core.base.entity.AccountEntity;
import com.seaskyland.llm.workflow.core.base.entity.AgentSchemaEntity;
import com.seaskyland.llm.workflow.core.base.entity.ApiKeyEntity;
import com.seaskyland.llm.workflow.core.base.entity.AppComponentEntity;
import com.seaskyland.llm.workflow.core.base.entity.AppEntity;
import com.seaskyland.llm.workflow.core.base.entity.AppVersionEntity;
import com.seaskyland.llm.workflow.core.base.entity.DocumentEntity;
import com.seaskyland.llm.workflow.core.base.entity.KnowledgeBaseEntity;
import com.seaskyland.llm.workflow.core.base.entity.LimitEntity;
import com.seaskyland.llm.workflow.core.base.entity.McpServerEntity;
import com.seaskyland.llm.workflow.core.base.entity.ModelEntity;
import com.seaskyland.llm.workflow.core.base.entity.PluginEntity;
import com.seaskyland.llm.workflow.core.base.entity.ProviderEntity;
import com.seaskyland.llm.workflow.core.base.entity.ReferEntity;
import com.seaskyland.llm.workflow.core.base.entity.ToolEntity;
import com.seaskyland.llm.workflow.core.base.entity.WorkspaceEntity;
import com.seaskyland.llm.workflow.core.base.mapper.AccountMapper;
import com.seaskyland.llm.workflow.core.base.mapper.AgentSchemaMapper;
import com.seaskyland.llm.workflow.core.base.mapper.ApiKeyMapper;
import com.seaskyland.llm.workflow.core.base.mapper.AppComponentMapper;
import com.seaskyland.llm.workflow.core.base.mapper.AppMapper;
import com.seaskyland.llm.workflow.core.base.mapper.AppVersionMapper;
import com.seaskyland.llm.workflow.core.base.mapper.DocumentMapper;
import com.seaskyland.llm.workflow.core.base.mapper.KnowledgeBaseMapper;
import com.seaskyland.llm.workflow.core.base.mapper.McpServerMapper;
import com.seaskyland.llm.workflow.core.base.mapper.ModelMapper;
import com.seaskyland.llm.workflow.core.base.mapper.PluginMapper;
import com.seaskyland.llm.workflow.core.base.mapper.ProviderMapper;
import com.seaskyland.llm.workflow.core.base.mapper.ReferMapper;
import com.seaskyland.llm.workflow.core.base.mapper.ToolMapper;
import com.seaskyland.llm.workflow.core.base.mapper.WorkspaceMapper;
import org.springframework.aot.hint.MemberCategory;
import org.springframework.aot.hint.RuntimeHints;
import org.springframework.aot.hint.RuntimeHintsRegistrar;

/**
 * Native-image hints for MyBatis-Plus metadata, mapper proxies, and custom type handlers.
 */
public class MybatisPlusRuntimeHints implements RuntimeHintsRegistrar {

	static final List<Class<?>> MYBATIS_PLUS_ENTITY_TYPES = List.of(AccountEntity.class, AgentSchemaEntity.class,
			ApiKeyEntity.class, AppComponentEntity.class, AppEntity.class, AppVersionEntity.class, DocumentEntity.class,
			KnowledgeBaseEntity.class, LimitEntity.class, McpServerEntity.class, ModelEntity.class, PluginEntity.class,
			ProviderEntity.class, ReferEntity.class, ToolEntity.class, WorkspaceEntity.class);

	static final List<Class<?>> MYBATIS_PLUS_MAPPER_TYPES = List.of(AccountMapper.class, AgentSchemaMapper.class,
			ApiKeyMapper.class, AppComponentMapper.class, AppMapper.class, AppVersionMapper.class, DocumentMapper.class,
			KnowledgeBaseMapper.class, McpServerMapper.class, ModelMapper.class, PluginMapper.class, ProviderMapper.class,
			ReferMapper.class, ToolMapper.class, WorkspaceMapper.class);

	@Override
	public void registerHints(RuntimeHints hints, ClassLoader classLoader) {
		MYBATIS_PLUS_ENTITY_TYPES.forEach(entityType -> hints.reflection()
			.registerType(entityType, MemberCategory.INVOKE_DECLARED_CONSTRUCTORS, MemberCategory.INVOKE_PUBLIC_METHODS,
					MemberCategory.ACCESS_DECLARED_FIELDS));
		MYBATIS_PLUS_MAPPER_TYPES.forEach(mapperType -> hints.reflection()
			.registerType(mapperType, MemberCategory.INVOKE_PUBLIC_METHODS));
		hints.reflection().registerType(SQLiteDateTypeHandler.class, MemberCategory.INVOKE_DECLARED_CONSTRUCTORS);
		hints.resources().registerPattern("mapper/*.xml");
	}

}
