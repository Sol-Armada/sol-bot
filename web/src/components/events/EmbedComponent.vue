<template>
  <link href="https://fonts.googleapis.com/css?family=Lato" rel="stylesheet" />
  <div class="container">
    <div class="col-md-10 offset-md-1">
      <div class="card-deck">
        <div class="card">
          <div class="card-header">Embed Editor</div>
          <div class="card-block">
            <form id="form">
              <div class="form-group">
                <label>Basic settings</label>
                <input
                  class="form-control"
                  id="title"
                  type="text"
                  placeholder="title"
                />
              </div>
              <div class="form-group">
                <input
                  class="form-control"
                  id="description"
                  type="text"
                  placeholder="description"
                />
              </div>
              <div class="form-group">
                <input
                  class="form-control"
                  id="url"
                  type="text"
                  placeholder="url"
                />
              </div>
              <div class="form-group">
                <label>Color</label>
                <input class="form-control" id="color" type="color" />
              </div>
              <div class="form-group">
                <label>Thumbnail settings</label>
                <input
                  class="form-control"
                  id="icon"
                  type="text"
                  placeholder="icon"
                />
              </div>
              <div class="form-group">
                <label>Author settings</label>
                <input
                  class="form-control"
                  id="author_name"
                  type="text"
                  placeholder="author name"
                />
              </div>
              <div class="form-group">
                <input
                  class="form-control"
                  id="author_url"
                  type="text"
                  placeholder="author link"
                />
              </div>
              <div class="form-group">
                <input
                  class="form-control"
                  id="author_icon"
                  type="text"
                  placeholder="author icon"
                />
              </div>
              <div class="form-group">
                <label>Fields</label>
              </div>
              <div class="input-fields"></div>
              <div class="form-group">
                <label>Footer settings</label>
                <input
                  class="form-control"
                  id="footer"
                  type="text"
                  placeholder="footer text"
                />
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
    <br />
    <div class="col-md-8 offset-md-2">
      <div class="card">
        <div class="card-header">Embed Preview</div>
        <div class="card-block preview">
          <div class="wrapper">
            <div class="side-colored"></div>
            <div class="card embed">
              <div class="card-block">
                <div class="embed-inner"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <br />
    <div class="col-md-10 offset-md-1">
      <div class="card">
        <div class="card-header">Python code</div>
        <div class="card-block">
          <pre><code class="python source"></code></pre>
        </div>
      </div>
    </div>
  </div>
</template>
<style lang="scss" scoped>
.embed {
  border-radius: 0 3px 3px 0;
  background: rgba(46, 48, 54, 0.3);
  border-color: rgba(46, 48, 54, 0.6);
  display: flex;
  padding: 8px 10px;
  color: rgba(255, 255, 255, 0.6);
  font-size: 14px;

  .card-block {
    padding: 0;
    display: flex;
    margin-bottom: 10px;
  }
  a {
    color: #0096cf;
  }
  img.embed-thumb {
    max-height: 80px;
    max-width: 80px;
    border-radius: 3px;
    flex-shrink: 0;
    width: auto;
    object-fit: contain;
    margin-left: 20px;
  }
  .embed-footer {
    font-size: 12px;
    span {
      color: rgba(255, 255, 255, 0.6);
    }
  }
  .embed-inner {
    .embed-title {
      color: #fff;
    }
    .embed-author {
      display: flex;
      align-items: center;
      margin-bottom: 5px;

      img.embed-author-icon {
        margin-right: 9px;
        width: 20px;
        height: 20px;
        object-fit: contain;
        border-radius: 50%;
      }
      .embed-author-name {
        display: inline-block;
        font-weight: 600;
        font-size: 14px;
        color: #fff !important;
      }
    }
    .fields {
      display: flex;
      flex-wrap: wrap;
      flex-direction: row;
      box-lines: miltiple;
      margin-top: -10px;

      .field {
        flex: 0;
        box-flex: 1;
        padding-top: 10px;
        max-width: 506px;
        min-width: 100%;

        &.inline {
          box-flex: 1;
          flex: 1;
          min-width: 150px;
          flex-basis: auto;
        }
        .field-name {
          color: #fff;
          font-size: 14px;
          margin-bottom: 4px;
          font-weight: 600;
        }
        .field-value {
          color: rgba(255, 255, 255, 0.7);
          font-size: 14px;
          font-weight: 500;
          line-height: 1.1em;
          white-space: pre-wrap;
          margin-top: 6px;
          word-wrap: break-word;
        }
      }
    }
  }
}
</style>
