/*
 * Copyright 2025 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.seaskyland.llm.workflow.core.base.service.impl;

import static com.seaskyland.llm.workflow.core.base.constants.CacheConstants.CACHE_ACCOUNT_UID_PREFIX;
import static com.seaskyland.llm.workflow.core.base.constants.CacheConstants.CACHE_EMPTY_ID;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.core.metadata.IPage;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.seaskyland.llm.workflow.core.base.constants.CacheConstants;
import com.seaskyland.llm.workflow.core.base.entity.AccountEntity;
import com.seaskyland.llm.workflow.core.base.manager.CacheManager;
import com.seaskyland.llm.workflow.core.base.manager.ModelManager;
import com.seaskyland.llm.workflow.core.base.manager.ProviderManager;
import com.seaskyland.llm.workflow.core.base.manager.TokenManager;
import com.seaskyland.llm.workflow.core.base.mapper.AccountMapper;
import com.seaskyland.llm.workflow.core.base.service.AccountService;
import com.seaskyland.llm.workflow.core.base.service.WorkspaceService;
import com.seaskyland.llm.workflow.core.config.JwtConfigProperties;
import com.seaskyland.llm.workflow.core.context.RequestContextHolder;
import com.seaskyland.llm.workflow.core.model.llm.domain.ModelConfigInfo;
import com.seaskyland.llm.workflow.core.model.llm.domain.ModelCredential;
import com.seaskyland.llm.workflow.core.model.llm.domain.ProviderConfigInfo;
import com.seaskyland.llm.workflow.core.utils.common.BeanCopierUtils;
import com.seaskyland.llm.workflow.core.utils.common.IdGenerator;
import com.seaskyland.llm.workflow.core.utils.security.PasswordCryptUtils;
import com.seaskyland.llm.workflow.runtime.domain.BaseQuery;
import com.seaskyland.llm.workflow.runtime.domain.PagingList;
import com.seaskyland.llm.workflow.runtime.domain.RequestContext;
import com.seaskyland.llm.workflow.runtime.domain.account.Account;
import com.seaskyland.llm.workflow.runtime.domain.account.ChangePasswordRequest;
import com.seaskyland.llm.workflow.runtime.domain.account.LoginRequest;
import com.seaskyland.llm.workflow.runtime.domain.account.Oauth2User;
import com.seaskyland.llm.workflow.runtime.domain.account.RefreshTokenRequest;
import com.seaskyland.llm.workflow.runtime.domain.account.TokenResponse;
import com.seaskyland.llm.workflow.runtime.domain.account.Workspace;
import com.seaskyland.llm.workflow.runtime.enums.AccountStatus;
import com.seaskyland.llm.workflow.runtime.enums.AccountType;
import com.seaskyland.llm.workflow.runtime.enums.ErrorCode;
import com.seaskyland.llm.workflow.runtime.exception.BizException;
import java.util.*;
import lombok.RequiredArgsConstructor;
import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Service;
import org.springframework.util.CollectionUtils;

/**
 * Account service implementation for managing user accounts and authentication. Handles account
 * operations including login, registration, and profile management.
 *
 * @since 1.0.0.3
 */
@Service
@RequiredArgsConstructor
public class AccountServiceImpl extends ServiceImpl<AccountMapper, AccountEntity>
    implements AccountService {

  /** Manages token generation and validation */
  private final TokenManager tokenManager;

  /** JWT configuration properties */
  private final JwtConfigProperties jwtConfigProperties;

  /** Redis cache manager for account data */
  private final CacheManager cacheManager;

  /** workspace service */
  private final WorkspaceService workspaceService;

  private final ProviderManager providerManager;

  /** model manager */
  private final ModelManager modelManager;

  /**
   * Authenticates user and generates access tokens
   *
   * @param loginRequest Login credentials
   * @return Token response containing access and refresh tokens
   */
  @Override
  public TokenResponse login(LoginRequest loginRequest) {
    AccountEntity accountEntity = getAccountByName(loginRequest.getUsername());
    if (accountEntity == null) {
      throw new BizException(ErrorCode.ACCOUNT_LOGIN_ERROR.toError());
    }

    if (!PasswordCryptUtils.match(loginRequest.getPassword(), accountEntity.getPassword())) {
      throw new BizException(ErrorCode.ACCOUNT_LOGIN_ERROR.toError());
    }

    accountEntity.setGmtLastLogin(new Date());
    this.updateById(accountEntity);

    // cache it
    Workspace workspace = workspaceService.getDefaultWorkspace(accountEntity.getAccountId());
    if (workspace == null) {
      throw new BizException(ErrorCode.DEFAULT_WORKSPACE_NOT_FOUND.toError());
    }

    accountEntity.setDefaultWorkspaceId(workspace.getWorkspaceId());
    String key = getAccountCacheKey(accountEntity.getAccountId());
    cacheManager.put(key, accountEntity);

    String accountId = accountEntity.getAccountId();
    return createTokenResponse(accountId);
  }

  /**
   * Refreshes access token using refresh token
   *
   * @param refreshTokenRequest Refresh token request
   * @return New token response
   */
  @Override
  public TokenResponse refreshToken(RefreshTokenRequest refreshTokenRequest) {
    String accountId =
        tokenManager.getAccountIdFromRefreshToken(refreshTokenRequest.getRefreshToken());
    if (accountId == null) {
      throw new BizException(ErrorCode.INVALID_REFRESH_TOKEN.toError());
    }

    TokenResponse response = createTokenResponse(accountId);
    tokenManager.deleteRefreshToken(refreshTokenRequest.getRefreshToken());
    return response;
  }

  @Override
  public TokenResponse login(Oauth2User oauth2User) {
    if (oauth2User == null || StringUtils.isBlank(oauth2User.getUserId())) {
      throw new BizException(ErrorCode.OAUTH2_USER_NOT_FOUND.toError());
    }

    String accountId;
    AccountEntity accountEntity = getAccountByName(oauth2User.getUserId());
    if (accountEntity == null) {
      Account account = new Account();
      account.setUsername(oauth2User.getUserId());
      account.setNickname(oauth2User.getName());
      account.setEmail(oauth2User.getEmail());
      account.setIcon(oauth2User.getIcon());
      account.setPassword(IdGenerator.uuid32());

      accountId = registerAccount(account);
      accountEntity = getAccountById(accountId);
    } else {
      accountId = accountEntity.getAccountId();
    }

    if (accountEntity == null) {
      throw new BizException(ErrorCode.ACCOUNT_NOT_FOUND.toError());
    }

    // cache it
    Workspace workspace = workspaceService.getDefaultWorkspace(accountEntity.getAccountId());
    if (workspace == null) {
      throw new BizException(ErrorCode.DEFAULT_WORKSPACE_NOT_FOUND.toError());
    }

    accountEntity.setDefaultWorkspaceId(workspace.getWorkspaceId());
    String key = getAccountCacheKey(accountEntity.getAccountId());
    cacheManager.put(key, accountEntity);

    return createTokenResponse(accountId);
  }

  /**
   * Invalidates access token on logout
   *
   * @param accessToken Token to invalidate
   */
  @Override
  public void logout(String accessToken) {
    tokenManager.deleteAccessToken(accessToken);
  }

  @Override
  public String registerAccount(Account account) {
    // check if account name exists
    AccountEntity accountEntity = getAccountByName(account.getUsername());
    if (accountEntity != null) {
      throw new BizException(ErrorCode.ACCOUNT_NAME_EXISTS.toError());
    }

    String accountId = IdGenerator.idStr();

    AccountEntity entity = BeanCopierUtils.copy(account, AccountEntity.class);
    entity.setAccountId(accountId);
    entity.setStatus(AccountStatus.NORMAL);
    entity.setType(AccountType.USER);
    entity.setPassword(PasswordCryptUtils.encode(account.getPassword()));
    entity.setEmail(account.getEmail());
    entity.setMobile(account.getMobile());
    entity.setGmtCreate(new Date());
    entity.setGmtModified(new Date());
    entity.setCreator(accountId);
    entity.setModifier(accountId);

    this.save(entity);

    // create workspace for account
    String workspaceId = createWorkspace(accountId);
    entity.setDefaultWorkspaceId(workspaceId);

    // cache it
    String key = getAccountCacheKey(accountId);
    cacheManager.put(key, entity);

    initAccountData(workspaceId, accountId);

    return accountId;
  }

  private void initAccountData(String workspaceId, String accountId) {
    // 初始化通义千问提供商
    ProviderConfigInfo providerConfigInfo = new ProviderConfigInfo();
    providerConfigInfo.setProvider("OpenAI");
    providerConfigInfo.setName("OpenAI");
    providerConfigInfo.setDescription("OpenAI");
    providerConfigInfo.setEnable(true);
    providerConfigInfo.setSource("preset");
    providerConfigInfo.setProtocol("OpenAI");
    providerConfigInfo.setSupportedModelTypes(Arrays.asList("llm", "text_embedding"));

    ModelCredential credential = new ModelCredential();
    credential.setEndpoint("https://api.openai.com");
    providerConfigInfo.setCredential(credential);

    providerManager.addProvider(providerConfigInfo);

    // 初始化通义千问模型
    List<ModelConfigInfo> models =
        Arrays.asList(
            createModelConfig(
                "gpt-4.1",
                "gpt-4.1",
                "llm",
                "chat",
                Arrays.asList("vision", "function_call", "reasoning")),
            createModelConfig(
                "gpt-4.1-mini",
                "gpt-4.1-mini",
                "llm",
                "chat",
                Arrays.asList("vision", "function_call", "reasoning")),
            createModelConfig(
                "gpt-4.1-nano",
                "gpt-4.1-nano",
                "llm",
                "chat",
                Arrays.asList("function_call", "reasoning")),
            createModelConfig(
                "gpt-4o", "gpt-4o", "llm", "chat", Arrays.asList("vision", "function_call")),
            createModelConfig(
                "gpt-4o-mini",
                "gpt-4o-mini",
                "llm",
                "chat",
                Arrays.asList("vision", "function_call")),
            createModelConfig(
                "text-embedding-3-small",
                "text-embedding-3-small",
                "text_embedding",
                "chat",
                Arrays.asList("embedding")),
            createModelConfig(
                "text-embedding-3-large",
                "text-embedding-3-large",
                "text_embedding",
                "chat",
                Arrays.asList("embedding")));
    // LLM 模型

    // 文本嵌入模型

    // 重排序模型

    for (ModelConfigInfo model : models) {
      model.setProvider("OpenAI");
      model.setSource("preset");
      model.setEnable(true);
      modelManager.addModel(model);
    }
  }

  private ModelConfigInfo createModelConfig(
      String modelId, String name, String type, String mode, List<String> tags) {
    ModelConfigInfo modelConfigInfo = new ModelConfigInfo();
    modelConfigInfo.setModelId(modelId);
    modelConfigInfo.setName(name);
    modelConfigInfo.setType(type);
    modelConfigInfo.setMode(mode);
    modelConfigInfo.setTags(tags);
    return modelConfigInfo;
  }

  /**
   * Creates a new user account
   *
   * @param account Account details
   * @return Generated account ID
   */
  @Override
  public String createAccount(Account account) {
    RequestContext context = RequestContextHolder.getRequestContext();
    checkAdminPermission(context.getAccountId());

    // check if account name exists
    AccountEntity accountEntity = getAccountByName(account.getUsername());
    if (accountEntity != null) {
      throw new BizException(ErrorCode.ACCOUNT_NAME_EXISTS.toError());
    }

    String accountId = IdGenerator.idStr();

    AccountEntity entity = BeanCopierUtils.copy(account, AccountEntity.class);
    entity.setAccountId(accountId);
    entity.setStatus(AccountStatus.NORMAL);
    entity.setType(AccountType.USER);
    entity.setPassword(PasswordCryptUtils.encode(account.getPassword()));
    entity.setEmail(account.getEmail());
    entity.setMobile(account.getMobile());
    entity.setGmtCreate(new Date());
    entity.setGmtModified(new Date());
    entity.setCreator(context.getAccountId());
    entity.setModifier(context.getAccountId());

    this.save(entity);

    // create workspace for account
    String workspaceId = createWorkspace(accountId);
    entity.setDefaultWorkspaceId(workspaceId);

    // cache it
    String key = getAccountCacheKey(accountId);
    cacheManager.put(key, entity);

    return accountId;
  }

  /**
   * Updates existing account information
   *
   * @param account Updated account details
   */
  @Override
  public void updateAccount(Account account) {
    RequestContext context = RequestContextHolder.getRequestContext();
    checkAdminPermission(context.getAccountId());

    AccountEntity entity = getAccountById(account.getAccountId());
    if (entity == null) {
      throw new BizException(ErrorCode.ACCOUNT_NOT_FOUND.toError());
    }

    if (StringUtils.isNotBlank(account.getPassword())) {
      entity.setPassword(PasswordCryptUtils.encode(account.getPassword()));
    }

    entity.setNickname(account.getNickname());
    entity.setIcon(account.getIcon());
    entity.setEmail(account.getEmail());
    entity.setMobile(account.getMobile());
    entity.setModifier(context.getAccountId());
    entity.setGmtModified(new Date());

    this.updateById(entity);

    // cache it
    String key = getAccountCacheKey(account.getAccountId());
    cacheManager.put(key, entity);
  }

  /**
   * Soft deletes an account
   *
   * @param accountId ID of account to delete
   */
  @Override
  public void deleteAccount(String accountId) {
    RequestContext context = RequestContextHolder.getRequestContext();
    checkAdminPermission(context.getAccountId());

    // delete from db
    AccountEntity entity = getAccountById(accountId);
    if (entity == null) {
      return;
    }

    // delete account
    entity.setStatus(AccountStatus.DELETED);
    entity.setGmtModified(new Date());
    entity.setModifier(context.getAccountId());
    this.updateById(entity);

    // delete from cache
    String cacheKey = getAccountCacheKey(accountId);
    cacheManager.delete(cacheKey);
  }

  /**
   * Lists accounts with pagination
   *
   * @param query Query parameters
   * @return Paginated list of accounts
   */
  @Override
  public PagingList<Account> listAccounts(BaseQuery query) {
    RequestContext context = RequestContextHolder.getRequestContext();
    checkAdminPermission(context.getAccountId());

    LambdaQueryWrapper<AccountEntity> queryWrapper = new LambdaQueryWrapper<>();
    queryWrapper.eq(AccountEntity::getType, AccountType.USER.getType());
    queryWrapper.ne(AccountEntity::getStatus, AccountStatus.DELETED.getStatus());
    if (StringUtils.isNotBlank(query.getName())) {
      queryWrapper.like(AccountEntity::getUsername, query.getName());
    }
    queryWrapper.orderByDesc(AccountEntity::getId);

    Page<AccountEntity> page = new Page<>(query.getCurrent(), query.getSize());
    IPage<AccountEntity> pageResult = this.page(page, queryWrapper);

    List<Account> accounts;
    if (CollectionUtils.isEmpty(pageResult.getRecords())) {
      accounts = new ArrayList<>();
    } else {
      accounts = pageResult.getRecords().stream().map(this::toAccountDTO).toList();
    }

    return new PagingList<>(query.getCurrent(), query.getSize(), pageResult.getTotal(), accounts);
  }

  /**
   * Retrieves account by ID
   *
   * @param accountId Account ID
   * @return Account details
   */
  @Override
  public Account getAccount(String accountId) {
    // RequestContext context = RequestContextHolder.getRequestContext();
    // checkAdminPermission(context.getAccountId());

    AccountEntity entity = getAccountById(accountId);
    if (entity == null) {
      return null;
    }

    return toAccountDTO(entity);
  }

  /**
   * Changes user password
   *
   * @param request Password change request
   */
  @Override
  public void changePassword(ChangePasswordRequest request) {
    RequestContext context = RequestContextHolder.getRequestContext();
    AccountEntity entity = getAccountById(context.getAccountId());

    if (Objects.isNull(entity)) {
      throw new BizException(ErrorCode.ACCOUNT_NOT_FOUND.toError());
    }

    if (!PasswordCryptUtils.match(request.getPassword(), entity.getPassword())) {
      throw new BizException(ErrorCode.ACCOUNT_PASSWORD_NOT_MATCH.toError());
    }

    String newEncodedPassword = PasswordCryptUtils.encode(request.getNewPassword());
    entity.setPassword(newEncodedPassword);
    entity.setGmtModified(new Date());
    this.updateById(entity);

    String key = getAccountCacheKey(context.getAccountId());
    cacheManager.put(key, entity);
  }

  /**
   * Gets current user's profile
   *
   * @return Account profile
   */
  @Override
  public Account getAccountProfile() {
    RequestContext context = RequestContextHolder.getRequestContext();
    AccountEntity entity = getAccountById(context.getAccountId());

    if (Objects.isNull(entity)) {
      throw new BizException(ErrorCode.ACCOUNT_NOT_FOUND.toError());
    }

    return toAccountDTO(entity);
  }

  /**
   * Creates token response with access and refresh tokens
   *
   * @param accountId Account ID
   * @return Token response
   */
  private TokenResponse createTokenResponse(String accountId) {
    String accessToken = tokenManager.generateAccessToken(accountId);
    String refreshToken = tokenManager.generateRefreshToken(accountId);

    return TokenResponse.builder()
        .accessToken(accessToken)
        .refreshToken(refreshToken)
        .expiresIn(
            System.currentTimeMillis() / 1000L + jwtConfigProperties.getAccessTokenExpiration())
        .build();
  }

  /**
   * Retrieves account by username
   *
   * @param username Username
   * @return Account entity
   */
  private AccountEntity getAccountByName(String username) {
    LambdaQueryWrapper<AccountEntity> queryWrapper = new LambdaQueryWrapper<>();
    queryWrapper
        .eq(AccountEntity::getUsername, username)
        .ne(AccountEntity::getStatus, AccountStatus.DELETED.getStatus());

    return this.getOne(queryWrapper);
  }

  /**
   * Retrieves account by ID with caching
   *
   * @param accountId Account ID
   * @return Account entity
   */
  private AccountEntity getAccountById(String accountId) {
    String key = getAccountCacheKey(accountId);
    AccountEntity entity = cacheManager.get(key);
    if (entity != null) {
      if (CACHE_EMPTY_ID.equals(entity.getId())) {
        return null;
      }

      return entity;
    }

    LambdaQueryWrapper<AccountEntity> queryWrapper = new LambdaQueryWrapper<>();
    queryWrapper
        .eq(AccountEntity::getAccountId, accountId)
        .ne(AccountEntity::getStatus, AccountStatus.DELETED.getStatus());

    Optional<AccountEntity> entityOptional = this.getOneOpt(queryWrapper);
    if (entityOptional.isEmpty()) {
      entity = new AccountEntity();
      entity.setId(CACHE_EMPTY_ID);
      cacheManager.put(key, entity, CacheConstants.CACHE_EMPTY_TTL);
      return null;
    }

    entity = entityOptional.get();

    // get default workspace
    Workspace workspace = workspaceService.getDefaultWorkspace(accountId);
    if (workspace == null) {
      throw new BizException(ErrorCode.DEFAULT_WORKSPACE_NOT_FOUND.toError());
    }

    entity.setDefaultWorkspaceId(workspace.getWorkspaceId());
    cacheManager.put(key, entity);
    return entity;
  }

  /**
   * Generates cache key for account
   *
   * @param accountId Account ID
   * @return Cache key
   */
  public static String getAccountCacheKey(String accountId) {
    return String.format(CACHE_ACCOUNT_UID_PREFIX, accountId);
  }

  /**
   * Converts account entity to DTO
   *
   * @param entity Account entity
   * @return Account DTO
   */
  private Account toAccountDTO(AccountEntity entity) {
    if (entity == null) {
      return null;
    }

    Account account = BeanCopierUtils.copy(entity, Account.class);
    account.setPassword(null);
    return account;
  }

  /**
   * Verifies admin permissions
   *
   * @param accountId Account ID to check
   */
  private void checkAdminPermission(String accountId) {
    AccountEntity entity = getAccountById(accountId);
    if (entity == null) {
      throw new BizException(ErrorCode.ACCOUNT_NOT_FOUND.toError());
    }

    if (AccountType.ADMIN != entity.getType()) {
      throw new BizException(ErrorCode.PERMISSION_DENIED.toError());
    }
  }

  /**
   * create workspace for account
   *
   * @param accountId account id
   * @return workspace id
   */
  private String createWorkspace(String accountId) {
    Workspace workspace = new Workspace();
    workspace.setAccountId(accountId);
    workspace.setName("Default Workspace");
    workspace.setDescription("Default workspace");
    return workspaceService.createWorkspace(workspace);
  }
}
