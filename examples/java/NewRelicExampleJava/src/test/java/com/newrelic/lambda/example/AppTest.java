package com.newrelic.lambda.example;

import com.amazonaws.services.lambda.runtime.ClientContext;
import com.amazonaws.services.lambda.runtime.CognitoIdentity;
import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.LambdaLogger;
import org.junit.Test;

import java.util.Collections;

import static org.junit.Assert.assertEquals;

public class AppTest {
  @Test
  public void successfulResponse() {
    App app = new App();
    String result = app.handleRequest(Collections.singletonMap("test", "case"), createContext());
    assertEquals("Success!", result);
  }

  private Context createContext() {
    return new Context() {
      @Override
      public String getAwsRequestId() {
        return "123";
      }

      @Override
      public String getLogGroupName() {
        return "logGroupName";
      }

      @Override
      public String getLogStreamName() {
        return "getLogStreamName";
      }

      @Override
      public String getFunctionName() {
        return "test-function";
      }

      @Override
      public String getFunctionVersion() {
        return "LATEST";
      }

      @Override
      public String getInvokedFunctionArn() {
        return "arn";
      }

      @Override
      public CognitoIdentity getIdentity() {
        return new CognitoIdentity() {
          @Override
          public String getIdentityId() {
            return "identity";
          }

          @Override
          public String getIdentityPoolId() {
            return "identityPoolId";
          }
        };
      }

      @Override
      public ClientContext getClientContext() {
        return null;
      }

      @Override
      public int getRemainingTimeInMillis() {
        return 100;
      }

      @Override
      public int getMemoryLimitInMB() {
        return 510;
      }

      @Override
      public LambdaLogger getLogger() {
        return new LambdaLogger() {
          @Override
          public void log(String string) {}

          @Override
          public void log(byte[] bytes) {}
        };
      }
    };
  }
}
