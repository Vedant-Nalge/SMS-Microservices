package com.sms.sender.service;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

@Slf4j
@Service
@RequiredArgsConstructor
public class BlockListService {

    private final RedisTemplate<String, String> redisTemplate;

    @Value("${app.redis.blocked-users-key}")
    private String blockedUsersKey;

    /**
     * Checks if a userId is on the block list.
     * The block list is a Redis Set keyed by blockedUsersKey.
     */
    public boolean isBlocked(String userId) {
        try {
            Boolean isMember = redisTemplate.opsForSet().isMember(blockedUsersKey, userId);
            boolean blocked = Boolean.TRUE.equals(isMember);
            if (blocked) {
                log.warn("UserId {} is on the block list — SMS will not be sent", userId);
            }
            return blocked;
        } catch (Exception e) {
            // Fail open: if Redis is unavailable, allow the message to proceed
            log.error("Redis error checking block list for userId={}: {}. Failing open.", userId, e.getMessage());
            return false;
        }
    }

    /** Adds a userId to the block list (admin/test helper). */
    public void blockUser(String userId) {
        redisTemplate.opsForSet().add(blockedUsersKey, userId);
        log.info("UserId {} added to block list", userId);
    }

    /** Removes a userId from the block list (admin/test helper). */
    public void unblockUser(String userId) {
        redisTemplate.opsForSet().remove(blockedUsersKey, userId);
        log.info("UserId {} removed from block list", userId);
    }
}
