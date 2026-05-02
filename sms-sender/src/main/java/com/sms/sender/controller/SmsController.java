package com.sms.sender.controller;

import com.sms.sender.model.Models;
import com.sms.sender.service.SmsService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

@Slf4j
@RestController
@RequestMapping("/v1/sms")
@RequiredArgsConstructor
public class SmsController {

    private final SmsService smsService;

    /**
     * POST /v1/sms/send
     * Accepts a send-SMS request, validates it, and kicks off the send flow.
     */
    @PostMapping("/send")
    public ResponseEntity<Models.SendSmsResponse> sendSms(
            @Valid @RequestBody Models.SendSmsRequest request) {

        log.info("Received POST /v1/sms/send for userId={}", request.getUserId());
        Models.SendSmsResponse response = smsService.sendSms(request);

        HttpStatus httpStatus = switch (response.getStatus()) {
            case "SUCCESS" -> HttpStatus.OK;
            case "BLOCKED" -> HttpStatus.FORBIDDEN;
            default -> HttpStatus.BAD_GATEWAY;  // vendor failure
        };

        return ResponseEntity.status(httpStatus).body(response);
    }

    /** Simple health-check endpoint. */
    @GetMapping("/health")
    public ResponseEntity<Map<String, String>> health() {
        return ResponseEntity.ok(Map.of("status", "UP", "service", "sms-sender"));
    }
}
