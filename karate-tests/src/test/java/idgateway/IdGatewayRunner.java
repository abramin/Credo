package idgateway;

import com.intuit.karate.junit5.Karate;

class IdGatewayRunner {

    @Karate.Test
    Karate testAll() {
        return Karate.run().relativeTo(getClass());
    }

    @Karate.Test
    Karate testAuth() {
        return Karate.run("auth").relativeTo(getClass());
    }

    @Karate.Test
    Karate testNormalFlow() {
        return Karate.run("auth/normal_flow").relativeTo(getClass());
    }

    @Karate.Test
    Karate testAttackPaths() {
        return Karate.run("auth/attack_paths").relativeTo(getClass());
    }
}
