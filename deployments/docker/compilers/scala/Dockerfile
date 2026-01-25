# Scala protobuf compiler image
FROM spoke/compiler-base:latest

# Install Java (required for Scala)
RUN apt-get update && \
    apt-get install -y openjdk-17-jdk && \
    rm -rf /var/lib/apt/lists/*

# Install sbt (Scala build tool)
RUN echo "deb https://repo.scala-sbt.org/scalasbt/debian all main" | tee /etc/apt/sources.list.d/sbt.list && \
    echo "deb https://repo.scala-sbt.org/scalasbt/debian /" | tee /etc/apt/sources.list.d/sbt_old.list && \
    curl -sL "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x2EE0EA64E40A89B84B2DF73499E82A75642AC823" | apt-key add && \
    apt-get update && \
    apt-get install -y sbt && \
    rm -rf /var/lib/apt/lists/*

# Install Scala
RUN wget https://downloads.lightbend.com/scala/2.13.12/scala-2.13.12.tgz && \
    tar -xzf scala-2.13.12.tgz && \
    mv scala-2.13.12 /usr/local/scala && \
    rm scala-2.13.12.tgz

ENV PATH="/usr/local/scala/bin:${PATH}"

# Install protoc-gen-scala (ScalaPB)
# We'll use coursier to install it
RUN curl -fL https://github.com/coursier/launchers/raw/master/cs-x86_64-pc-linux.gz | gzip -d > /usr/local/bin/cs && \
    chmod +x /usr/local/bin/cs && \
    cs install scalapb-plugin --install-dir /usr/local/bin

# Create symlink for protoc-gen-scala
RUN ln -s /usr/local/bin/scalapb-plugin /usr/local/bin/protoc-gen-scala

# Verify installation
RUN protoc --version && \
    scala -version && \
    sbt --version && \
    which protoc-gen-scala

WORKDIR /workspace

CMD ["protoc", "--version"]
